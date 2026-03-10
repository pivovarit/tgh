package ui

import (
	"strconv"
	"strings"

	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"
	"github.com/pivovarit/tgh/internal/gh"
)

const (
	selW     = 1
	ageW     = 7
	ciW      = 4
	overhead = 12
)

type colWidths struct {
	repo, title, author int
}

func computeColWidths(prs []gh.PR, width int) colWidths {
	repoW, titleW, authorW := 4, 5, 6
	for _, pr := range prs {
		if w := len([]rune(pr.Repository.NameWithOwner)); w > repoW {
			repoW = w
		}
		if w := len([]rune(pr.Title)); w > titleW {
			titleW = w
		}
		if w := len([]rune(pr.Author.Login)); w > authorW {
			authorW = w
		}
	}

	repoW = min(repoW, 30)
	titleW = min(titleW, 60)
	authorW = min(authorW, 20)

	remaining := width - selW - ageW - ciW - overhead
	if remaining < 30 {
		remaining = 30
	}

	total := repoW + titleW + authorW
	if total > 0 {
		repoW = max(remaining*repoW/total, 4)
		authorW = min(max(remaining*authorW/total, 6), 20)
		titleW = max(remaining-repoW-authorW, 5)
	}
	return colWidths{repoW, titleW, authorW}
}

func buildRows(prs []gh.PR, checkStatus map[gh.PRKey]string, reviewStatus map[gh.PRKey]gh.ReviewSummary, mergeState map[gh.PRKey]string, autoMerge map[gh.PRKey]bool, selected map[gh.PRKey]bool, w colWidths) []table.Row {
	rows := make([]table.Row, len(prs))
	for i, pr := range prs {
		key := keyFor(pr)
		sel := " "
		if selected[key] {
			sel = "●"
		}
		rows[i] = table.Row{
			sel,
			trunc(pr.Repository.NameWithOwner, w.repo),
			trunc(prNum(pr.Number)+" "+pr.Title, w.title),
			trunc(gh.RelativeTime(pr.CreatedAt), ageW),
			statusSymbol(checkStatus[key], reviewStatus[key], mergeState[key], autoMerge[key]),
			trunc(pr.Author.Login, w.author),
		}
	}
	return rows
}

func buildTable(prs []gh.PR, checkStatus map[gh.PRKey]string, reviewStatus map[gh.PRKey]gh.ReviewSummary, mergeState map[gh.PRKey]string, selected map[gh.PRKey]bool, width int) (table.Model, colWidths) {
	innerWidth := width - 2
	w := computeColWidths(prs, innerWidth)

	cols := []table.Column{
		{Title: "", Width: selW},
		{Title: "Repo", Width: w.repo},
		{Title: "Title", Width: w.title},
		{Title: "Age", Width: ageW},
		{Title: "CI", Width: ciW},
		{Title: "Author", Width: w.author},
	}

	var rows []table.Row
	if len(prs) > 0 {
		rows = buildRows(prs, checkStatus, reviewStatus, mergeState, nil, selected, w)
	}
	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithWidth(innerWidth),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#0369A1")).
		BorderBottom(true).
		Bold(true).
		Foreground(lipgloss.Color("#38BDF8"))
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#F0F9FF")).
		Background(lipgloss.Color("#0369A1")).
		Bold(false)
	t.SetStyles(s)

	return t, w
}

func statusSymbol(ci string, rev gh.ReviewSummary, merge string, autoMerge bool) string {
	var ciSym string
	switch ci {
	case "success":
		ciSym = checkSuccessStyle.Render("✓")
	case "failure":
		ciSym = checkFailureStyle.Render("✗")
	case "pending":
		ciSym = checkPendingStyle.Render("○")
	case "none":
		ciSym = checkNoneStyle.Render("–")
	default:
		ciSym = " "
	}

	var revSym string
	if rev.ChangesRequested > 0 {
		revSym = reviewChangesStyle.Render("±")
	} else if rev.Approvals > 0 {
		s := strconv.Itoa(rev.Approvals)
		if rev.Approvals > 9 {
			s = "9+"
		}
		revSym = reviewApprovedStyle.Render(s)
	}

	var mergeSym string
	switch merge {
	case "BEHIND":
		mergeSym = behindStyle.Render("↓")
	case "DIRTY":
		mergeSym = conflictStyle.Render("!")
	}

	result := ciSym
	if revSym != "" {
		result += revSym
	}
	if mergeSym != "" {
		result += mergeSym
	}
	if autoMerge {
		result += autoMergeStyle.Render("»")
	}
	return result
}

func prNum(n int) string {
	return "#" + strconv.Itoa(n)
}

func trunc(s string, max int) string {
	if max < 1 {
		return ""
	}
	s = strings.TrimPrefix(s, "/")
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max <= 1 {
		return "…"
	}
	return string(runes[:max-1]) + "…"
}
