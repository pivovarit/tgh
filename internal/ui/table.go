package ui

import (
	"strings"

	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"
	"github.com/pivovarit/tgh/internal/gh"
)

const (
	selW     = 1
	numW     = 6
	ageW     = 7
	ciW      = 4
	overhead = 15
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

	remaining := width - selW - numW - ageW - ciW - overhead
	if remaining < 30 {
		remaining = 30
	}

	total := repoW + titleW + authorW
	if total > 0 {
		repoW = max(remaining*repoW/total, 4)
		titleW = max(remaining*titleW/total, 5)
		authorW = max(remaining-repoW-titleW, 6)
	}
	return colWidths{repoW, titleW, authorW}
}

func buildRows(prs []gh.PR, checkStatus map[int]string, reviewStatus map[int]gh.ReviewSummary, selected map[prKey]bool, w colWidths) []table.Row {
	rows := make([]table.Row, len(prs))
	for i, pr := range prs {
		sel := " "
		if selected[keyFor(pr)] {
			sel = "●"
		}
		rows[i] = table.Row{
			sel,
			trunc(pr.Repository.NameWithOwner, w.repo),
			trunc(prNum(pr.Number), numW),
			trunc(pr.Title, w.title),
			trunc(pr.Author.Login, w.author),
			trunc(gh.RelativeTime(pr.CreatedAt), ageW),
			statusSymbol(checkStatus[pr.Number], reviewStatus[pr.Number]),
		}
	}
	return rows
}

func buildTable(prs []gh.PR, checkStatus map[int]string, reviewStatus map[int]gh.ReviewSummary, selected map[prKey]bool, width int) (table.Model, colWidths) {
	w := computeColWidths(prs, width)

	cols := []table.Column{
		{Title: "", Width: selW},
		{Title: "Repo", Width: w.repo},
		{Title: "#", Width: numW},
		{Title: "Title", Width: w.title},
		{Title: "Author", Width: w.author},
		{Title: "Age", Width: ageW},
		{Title: "CI", Width: ciW},
	}

	t := table.New(
		table.WithColumns(cols),
		table.WithRows(buildRows(prs, checkStatus, reviewStatus, selected, w)),
		table.WithFocused(true),
		table.WithWidth(width),
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

func statusSymbol(ci string, rev gh.ReviewSummary) string {
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
		revSym = reviewApprovedStyle.Render(itoa(rev.Approvals))
	}

	if revSym != "" {
		return ciSym + revSym
	}
	return ciSym
}

func prNum(n int) string {
	return "#" + itoa(n)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	b := make([]byte, 0, 10)
	if n < 0 {
		b = append(b, '-')
		n = -n
	}
	var digits [10]byte
	pos := 0
	for n > 0 {
		digits[pos] = byte('0' + n%10)
		pos++
		n /= 10
	}
	for pos > 0 {
		pos--
		b = append(b, digits[pos])
	}
	return string(b)
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
