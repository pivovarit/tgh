package ui

import (
	"slices"
	"strings"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"github.com/pivovarit/tgh/internal/gh"
)

const (
	chromeTitle        = 1
	chromeTitleMargin  = 1
	chromeTitleNewline = 1
	chromeBorderTop    = 1
	chromeBorderBottom = 1
	chromeHelpNewline  = 1
	chromeHelpMargin   = 1
	chromeHelp         = 1

	tableChrome = chromeTitle + chromeTitleMargin + chromeTitleNewline +
		chromeBorderTop + chromeBorderBottom +
		chromeHelpNewline + chromeHelpMargin + chromeHelp
)

type prKey struct {
	Num  int
	Repo string
}

func keyFor(pr gh.PR) prKey {
	return prKey{Num: pr.Number, Repo: pr.Repository.NameWithOwner}
}

type Operation int

const (
	OpNone Operation = iota
	OpConfirmApprove
	OpConfirmClose
	OpConfirmMerge
	OpConfirmUpdate
	OpApproving
	OpClosing
	OpMerging
	OpUpdating
)

type App struct {
	client gh.Client
	owners []string
	table  table.Model
	prs    []gh.PR

	prMode       gh.PRMode
	loading      bool
	op           Operation
	confirmNum   int
	confirmTitle string
	confirmRepo  string

	filtering   bool
	filterQuery string

	selected     map[prKey]bool
	bulkPending  int
	bulkTotal    int
	bulkFailed   int
	bulkApproved []gh.PR

	err        error
	warnMsg    string
	copiedName string

	width  int
	height int

	viewportStart int

	detail       detailState
	prsByNumber  map[int]gh.PR
	checkStatus  map[int]string
	reviewStatus map[int]gh.ReviewSummary
	mergeState   map[int]string
	widths       colWidths
}

func New(owners []string) App {
	return newWithClient(gh.CLI{}, owners)
}

func newWithClient(c gh.Client, owners []string) App {
	t, w := buildTable(nil, nil, nil, nil, nil, 120)
	return App{
		client:       c,
		owners:       owners,
		loading:      true,
		table:        t,
		widths:       w,
		checkStatus:  map[int]string{},
		reviewStatus: map[int]gh.ReviewSummary{},
		mergeState:   map[int]string{},
	}
}

func (m App) Init() tea.Cmd {
	return m.client.FetchPRs(m.prMode, m.owners)
}

func (m App) filteredPRs() []gh.PR {
	if m.filterQuery == "" {
		return m.prs
	}
	q := strings.ToLower(m.filterQuery)
	match := func(pr gh.PR) bool {
		if after, ok := strings.CutPrefix(q, "repo:"); ok {
			return strings.Contains(strings.ToLower(pr.Repository.NameWithOwner), after)
		}
		if after, ok := strings.CutPrefix(q, "author:"); ok {
			return strings.Contains(strings.ToLower(pr.Author.Login), after)
		}
		return strings.Contains(strings.ToLower(pr.Title), q) ||
			strings.Contains(strings.ToLower(pr.Author.Login), q) ||
			strings.Contains(strings.ToLower(pr.Repository.NameWithOwner), q)
	}
	var out []gh.PR
	for _, pr := range m.prs {
		if match(pr) {
			out = append(out, pr)
		}
	}
	return out
}

func (m App) currentSelectedNumber() int {
	if pr := m.currentPR(); pr != nil {
		return pr.Number
	}
	return 0
}

func (m App) currentPR() *gh.PR {
	filtered := m.filteredPRs()
	if c := m.table.Cursor(); c >= 0 && c < len(filtered) {
		return &filtered[c]
	}
	return nil
}

func indexPRs(prs []gh.PR) map[int]gh.PR {
	idx := make(map[int]gh.PR, len(prs))
	for _, pr := range prs {
		idx[pr.Number] = pr
	}
	return idx
}

func (m App) selectedPRs() []gh.PR {
	if len(m.selected) == 0 {
		return nil
	}
	var out []gh.PR
	for _, pr := range m.prs {
		if m.selected[keyFor(pr)] {
			out = append(out, pr)
		}
	}
	return out
}

func (m App) toggleSelect() App {
	filtered := m.filteredPRs()
	cursor := m.table.Cursor()
	if cursor < 0 || cursor >= len(filtered) {
		return m
	}
	key := keyFor(filtered[cursor])
	if m.selected == nil {
		m.selected = make(map[prKey]bool)
	}
	if m.selected[key] {
		delete(m.selected, key)
	} else {
		m.selected[key] = true
	}
	m.table.SetRows(buildRows(filtered, m.checkStatus, m.reviewStatus, m.mergeState, m.selected, m.widths))
	return m
}

func (m App) refreshTableRows() App {
	m.table.SetRows(buildRows(m.filteredPRs(), m.checkStatus, m.reviewStatus, m.mergeState, m.selected, m.widths))
	return m
}

func (m App) removePR(num int, repo string) App {
	prevCursor := m.table.Cursor()
	var kept []gh.PR
	for _, pr := range m.prs {
		if !(pr.Number == num && pr.Repository.NameWithOwner == repo) {
			kept = append(kept, pr)
		}
	}
	m.prs = kept
	m.prsByNumber = indexPRs(m.prs)

	filtered := m.filteredPRs()
	m.table.SetRows(buildRows(filtered, m.checkStatus, m.reviewStatus, m.mergeState, m.selected, m.widths))

	cursor := min(prevCursor, len(filtered)-1)
	if cursor >= 0 {
		m.table.SetCursor(cursor)
		h := m.tableHeight()
		if cursor < m.viewportStart {
			m.viewportStart = cursor
		} else if h > 0 && cursor >= m.viewportStart+h {
			m.viewportStart = cursor - h + 1
		}
	}
	return m
}

func (m App) rebuildTable(selectedNum int) App {
	prevCursor := m.table.Cursor()
	filtered := m.filteredPRs()

	m.table, m.widths = buildTable(filtered, m.checkStatus, m.reviewStatus, m.mergeState, m.selected, m.width)
	m.table.SetHeight(m.tableHeight())
	m.viewportStart = 0

	cursor := -1
	if selectedNum > 0 {
		if _, ok := m.prsByNumber[selectedNum]; ok {
			cursor = slices.IndexFunc(filtered, func(pr gh.PR) bool { return pr.Number == selectedNum })
		}
	}
	if cursor < 0 && len(filtered) > 0 {
		cursor = min(prevCursor, len(filtered)-1)
	}
	if cursor >= 0 {
		m.table.SetCursor(cursor)
		if h := m.tableHeight(); h > 0 && cursor >= h {
			m.viewportStart = cursor - h + 1
		}
	}
	return m
}
