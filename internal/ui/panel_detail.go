package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/pivovarit/tgh/internal/gh"
)

const detailPanelHeight = 20

type detailState struct {
	visible  bool
	prNumber int
	prTitle  string
	pr       *gh.PRDetail
	lines    []string
	scroll   scrollState
	loading  bool
}

func (m App) closeDetail() App {
	m.detail.visible = false
	m.detail.pr = nil
	m.detail.lines = nil
	m.detail.scroll = scrollState{}
	m.detail.loading = false
	m.table.SetHeight(m.tableHeight())
	return m
}

func buildDetailLines(pr gh.PRDetail, width int) []string {
	var lines []string
	maxW := width - 4
	if maxW < 20 {
		maxW = 20
	}

	lines = append(lines, panelSectionStyle.Render("  Description"))
	if pr.Body == "" {
		lines = append(lines, panelLineStyle.Render("  (no description)"))
	} else {
		lines = append(lines, renderMarkdown(pr.Body, width)...)
	}

	lines = append(lines, "")
	lines = append(lines, panelLineStyle.Render(fmt.Sprintf(
		"  Changes: +%d -%d in %d file(s)",
		pr.Additions, pr.Deletions, pr.ChangedFiles,
	)))

	lines = append(lines, "")
	lines = append(lines, panelSectionStyle.Render("  Checks"))
	if len(pr.StatusCheckRollup) == 0 {
		lines = append(lines, panelLineStyle.Render("  (no checks)"))
	} else {
		for _, c := range pr.StatusCheckRollup {
			name := trunc(c.DisplayName(), maxW-12)
			state := c.DisplayState()
			stateStyled := styleCheckState(state)
			lines = append(lines, fmt.Sprintf("  %-*s %s", maxW-12, name, stateStyled))
		}
	}

	lines = append(lines, "")
	lines = append(lines, panelSectionStyle.Render("  Reviews"))
	if len(pr.LatestReviews) == 0 {
		lines = append(lines, panelLineStyle.Render("  (no reviews)"))
	} else {
		for _, r := range pr.LatestReviews {
			stateStyled := styleReviewState(r.State)
			lines = append(lines, fmt.Sprintf("  %-20s %s", trunc(r.Author.Login, 20), stateStyled))
		}
	}

	return lines
}

func renderMarkdown(body string, width int) []string {
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width-2),
	)
	if err == nil {
		if rendered, err := r.Render(body); err == nil {
			var out []string
			for _, l := range strings.Split(strings.TrimRight(rendered, "\n"), "\n") {
				out = append(out, l)
			}
			return out
		}
	}
	var out []string
	for _, l := range strings.Split(strings.TrimRight(body, "\n"), "\n") {
		out = append(out, panelLineStyle.Render("  "+l))
	}
	return out
}

func styleCheckState(state string) string {
	switch strings.ToUpper(state) {
	case "SUCCESS", "COMPLETED":
		return checkSuccessStyle.Render(state)
	case "FAILURE", "ACTION_REQUIRED", "CANCELLED", "TIMED_OUT":
		return checkFailureStyle.Render(state)
	default:
		return checkPendingStyle.Render(state)
	}
}

func styleReviewState(state string) string {
	switch state {
	case "APPROVED":
		return reviewApprovedStyle.Render(state)
	case "CHANGES_REQUESTED":
		return reviewChangesStyle.Render(state)
	default:
		return panelLineStyle.Render(state)
	}
}

func (m App) renderDetailPanel() string {
	var b strings.Builder
	b.WriteString(dividerStyle.Render(strings.Repeat("─", m.width)))
	b.WriteString("\n")

	title := fmt.Sprintf("  PR #%d – %s", m.detail.prNumber, trunc(m.detail.prTitle, m.width-20))
	b.WriteString(panelTitleStyle.Render(title))
	b.WriteString("\n")

	if m.detail.loading {
		b.WriteString(panelLineStyle.Render("  Loading…"))
		for i := 1; i < detailPanelHeight-2; i++ {
			b.WriteString("\n")
		}
		return b.String()
	}

	lines := m.detail.lines
	start := m.detail.scroll.offset
	end := start + detailPanelHeight - 2
	if end > len(lines) {
		end = len(lines)
	}

	shown := 0
	for i := start; i < end; i++ {
		b.WriteString(lines[i])
		b.WriteString("\n")
		shown++
	}
	for ; shown < detailPanelHeight-2; shown++ {
		b.WriteString("\n")
	}

	return b.String()
}
