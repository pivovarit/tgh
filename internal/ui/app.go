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

func keyFor(pr gh.PR) gh.PRKey {
	return gh.PRKey{Num: pr.Number, Repo: pr.Repository.NameWithOwner}
}

type Operation int

const (
	OpNone Operation = iota
	OpConfirmApprove
	OpConfirmClose
	OpConfirmMerge
	OpConfirmUpdate
	OpConfirmAutoMerge
	OpApproving
	OpClosing
	OpMerging
	OpUpdating
	OpAutoMerging
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

	selected     map[gh.PRKey]bool
	bulkPending  int
	bulkTotal    int
	bulkFailed   int
	bulkApproved []gh.PR

	err        error
	warnMsg    string
	copiedName string

	width  int
	height int

	cursor        int
	viewportStart int

	detail          detailState
	prsByKey        map[gh.PRKey]gh.PR
	checkStatus     map[gh.PRKey]string
	reviewStatus    map[gh.PRKey]gh.ReviewSummary
	mergeState      map[gh.PRKey]string
	autoMergeStatus map[gh.PRKey]bool
	widths          colWidths
}

func New(owners []string) App {
	return newWithClient(gh.CLI{}, owners)
}

func newWithClient(c gh.Client, owners []string) App {
	t, w := buildTable(nil, nil, nil, nil, nil, 120)
	return App{
		client:          c,
		owners:          owners,
		loading:         true,
		table:           t,
		widths:          w,
		prsByKey:        map[gh.PRKey]gh.PR{},
		checkStatus:     map[gh.PRKey]string{},
		reviewStatus:    map[gh.PRKey]gh.ReviewSummary{},
		mergeState:      map[gh.PRKey]string{},
		autoMergeStatus: map[gh.PRKey]bool{},
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

func (m App) currentSelectedKey() *gh.PRKey {
	if pr := m.currentPR(); pr != nil {
		k := keyFor(*pr)
		return &k
	}
	return nil
}

func (m App) currentPR() *gh.PR {
	filtered := m.filteredPRs()
	if m.cursor >= 0 && m.cursor < len(filtered) {
		return &filtered[m.cursor]
	}
	return nil
}

func indexPRs(prs []gh.PR) map[gh.PRKey]gh.PR {
	idx := make(map[gh.PRKey]gh.PR, len(prs))
	for _, pr := range prs {
		idx[keyFor(pr)] = pr
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
	if m.cursor < 0 || m.cursor >= len(filtered) {
		return m
	}
	key := keyFor(filtered[m.cursor])
	if m.selected == nil {
		m.selected = make(map[gh.PRKey]bool)
	}
	if m.selected[key] {
		delete(m.selected, key)
	} else {
		m.selected[key] = true
	}
	m = m.syncTableViewport()
	return m
}

func (m App) moveCursor(delta, n, height int) App {
	if n == 0 {
		return m
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	} else if m.cursor >= n {
		m.cursor = n - 1
	}
	if m.cursor < m.viewportStart {
		m.viewportStart = m.cursor
	} else if height > 0 && m.cursor >= m.viewportStart+height {
		m.viewportStart = m.cursor - height + 1
	}
	m = m.syncTableViewport()
	return m
}

func (m App) visibleRows() int {
	return m.table.Height()
}

func (m App) syncTableViewport() App {
	filtered := m.filteredPRs()
	if m.viewportStart >= len(filtered) {
		m.viewportStart = max(0, len(filtered)-1)
	}
	height := m.visibleRows()
	end := m.viewportStart + height
	if end > len(filtered) {
		end = len(filtered)
	}
	visible := filtered[m.viewportStart:end]
	m.table.SetRows(buildRows(visible, m.checkStatus, m.reviewStatus, m.mergeState, m.autoMergeStatus, m.selected, m.widths))
	m.table.SetCursor(m.cursor - m.viewportStart)
	return m
}

func (m App) refreshTableRows() App {
	return m.syncTableViewport()
}

func (m App) removePR(num int, repo string) App {
	var kept []gh.PR
	for _, pr := range m.prs {
		if !(pr.Number == num && pr.Repository.NameWithOwner == repo) {
			kept = append(kept, pr)
		}
	}
	m.prs = kept
	m.prsByKey = indexPRs(m.prs)

	filtered := m.filteredPRs()
	if m.cursor >= len(filtered) {
		m.cursor = len(filtered) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor < m.viewportStart {
		m.viewportStart = m.cursor
	}
	m = m.syncTableViewport()
	return m
}

func (m App) invalidateRepoStatuses(repo string) App {
	for k := range m.checkStatus {
		if k.Repo == repo {
			m.checkStatus[k] = "pending"
		}
	}
	for k := range m.mergeState {
		if k.Repo == repo {
			delete(m.mergeState, k)
		}
	}
	return m
}

func (m App) prsInRepo(repo string) []gh.PR {
	var result []gh.PR
	for _, pr := range m.prs {
		if pr.Repository.NameWithOwner == repo {
			result = append(result, pr)
		}
	}
	return result
}

func (m App) rebuildTable(selectedKey *gh.PRKey) App {
	filtered := m.filteredPRs()

	m.table, m.widths = buildTable(nil, nil, nil, nil, nil, m.width)
	m.table.SetHeight(m.tableHeight())
	m.viewportStart = 0

	newCursor := -1
	if selectedKey != nil {
		newCursor = slices.IndexFunc(filtered, func(pr gh.PR) bool {
			return pr.Number == selectedKey.Num && pr.Repository.NameWithOwner == selectedKey.Repo
		})
	}
	if newCursor < 0 && len(filtered) > 0 {
		newCursor = min(m.cursor, len(filtered)-1)
	}
	if newCursor < 0 {
		newCursor = 0
	}
	m.cursor = newCursor
	if h := m.visibleRows(); h > 0 && m.cursor >= h {
		m.viewportStart = m.cursor - h + 1
	}
	m = m.syncTableViewport()
	return m
}
