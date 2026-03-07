package ui

import (
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
)

func (m App) View() tea.View {
	var b strings.Builder

	mode := m.prMode.Label()

	sep := titleHintStyle.Render("  ·  ")
	styledLeft := titleStyle.Render(" tgh") + sep +
		titleStyle.Render(mode) + titleHintStyle.Render(" [A]") + sep +
		titleHintStyle.Render("/ filter") + sep +
		titleHintStyle.Render("r refresh")
	if m.filterQuery != "" {
		styledLeft += titleHintStyle.Render(": ") + titleStyle.Render(fmt.Sprintf("%q", m.filterQuery))
	}
	if n := len(m.selected); n > 0 {
		styledLeft += sep + confirmStyle.Render(fmt.Sprintf("✦ %d selected", n))
	}

	b.WriteString(styledLeft)
	b.WriteString("\n\n")

	filtered := m.filteredPRs()

	switch {
	case m.loading && len(m.prs) == 0:
		b.WriteString(emptyStyle.Render("Fetching pull requests…"))

	case m.err != nil:
		b.WriteString(errorStyle.Render("  Error: " + m.err.Error()))
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("  Press " + keyStyle.Render("r") + " to retry, " + keyStyle.Render("q") + " to quit."))
		v := tea.NewView(b.String())
		v.AltScreen = true
		return v

	case len(m.prs) == 0:
		b.WriteString(emptyStyle.Render(fmt.Sprintf("No pull requests found for %q.", m.prMode.Label())))
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("  Press " +
			keyStyle.Render("A") + " to toggle all PRs, " +
			keyStyle.Render("r") + " to refresh, " +
			keyStyle.Render("q") + " to quit."))
		v := tea.NewView(b.String())
		v.AltScreen = true
		return v

	case len(filtered) == 0:
		b.WriteString(emptyStyle.Render(fmt.Sprintf("No pull requests match %q.", m.filterQuery)))

	default:
		const headerLines = 2

		lines := strings.Split(m.table.View(), "\n")
		cursor := m.table.Cursor()
		for i, line := range lines {
			dataIdx := i - headerLines
			if dataIdx < 0 {
				continue
			}
			prIdx := m.viewportStart + dataIdx
			if prIdx >= len(filtered) {
				break
			}
			if prIdx != cursor {
				pr := filtered[prIdx]
				switch pr.State {
				case "merged":
					lines[i] = mergedRowStyle.Render(line)
				case "closed":
					lines[i] = closedRowStyle.Render(line)
				}
			}
		}
		b.WriteString(tableStyle.Render(strings.Join(lines, "\n")))
	}

	if m.detail.visible {
		b.WriteString("\n")
		b.WriteString(m.renderDetailPanel())
	}

	b.WriteString("\n")
	b.WriteString(m.helpBar())

	v := tea.NewView(b.String())
	v.AltScreen = true
	return v
}

func confirmPrompt(action string, num int, title string, nSel int) string {
	if nSel > 0 {
		return confirmStyle.Render(fmt.Sprintf("  %s %d PRs? press ", action, nSel)) +
			keyStyle.Render("y") +
			confirmStyle.Render(" to confirm, ") +
			keyStyle.Render("n") +
			confirmStyle.Render(" to cancel")
	}
	return confirmStyle.Render("  "+action+" #"+strconv.Itoa(num)+" ") +
		confirmNameStyle.Render(trunc(title, 50)) +
		confirmStyle.Render("? press ") +
		keyStyle.Render("y") +
		confirmStyle.Render(" to confirm, ") +
		keyStyle.Render("n") +
		confirmStyle.Render(" to cancel")
}

func progressPrompt(action string, num int, bulkTotal, bulkPending int) string {
	if bulkPending > 0 {
		return confirmStyle.Render(fmt.Sprintf("  %s %d PRs… (%d remaining)", action, bulkTotal, bulkPending))
	}
	return confirmStyle.Render(fmt.Sprintf("  %s #%d…", action, num))
}

func (m App) helpBar() string {
	nSel := len(m.selected)
	switch {
	case m.op == OpApproving:
		return progressPrompt("Approving", m.confirmNum, m.bulkTotal, m.bulkPending)
	case m.op == OpClosing:
		return progressPrompt("Closing", m.confirmNum, m.bulkTotal, m.bulkPending)
	case m.op == OpMerging:
		return progressPrompt("Merging", m.confirmNum, m.bulkTotal, m.bulkPending)
	case m.op == OpUpdating:
		return confirmStyle.Render(fmt.Sprintf("  Updating branch for #%d…", m.confirmNum))
	case m.op == OpConfirmApprove:
		return confirmPrompt("Approve", m.confirmNum, m.confirmTitle, nSel)
	case m.op == OpConfirmClose:
		return confirmPrompt("Close", m.confirmNum, m.confirmTitle, nSel)
	case m.op == OpConfirmMerge:
		return confirmPrompt("Squash & merge", m.confirmNum, m.confirmTitle, nSel)
	case m.op == OpConfirmUpdate:
		return confirmPrompt("Update branch for", m.confirmNum, m.confirmTitle, 0)
	case m.detail.visible:
		return helpStyle.Render(
			"  ↑/↓ scroll · " +
				keyStyle.Render("g") + " top · " +
				keyStyle.Render("G") + " bottom · " +
				keyStyle.Render("esc") + "/" + keyStyle.Render("v") + " close · " +
				keyStyle.Render("q") + " quit",
		)
	case m.filtering:
		return helpStyle.Render(
			"  / " + keyStyle.Render(m.filterQuery+"▌") +
				"  · " + keyStyle.Render("repo:") + " · " + keyStyle.Render("author:") + " · esc/enter exit",
		)
	default:
		if m.warnMsg != "" {
			var styledMsg string
			switch {
			case strings.HasPrefix(m.warnMsg, "✓"):
				styledMsg = successStyle.Render(m.warnMsg)
			case strings.HasPrefix(m.warnMsg, "⚠"):
				styledMsg = warnStyle.Render(m.warnMsg)
			default:
				styledMsg = confirmStyle.Render(m.warnMsg)
			}
			return helpStyle.Render("  ") + styledMsg
		}
		if m.copiedName != "" {
			return helpStyle.Render(
				"  " + successStyle.Render("✓ copied URL of ") + keyStyle.Render(trunc(m.copiedName, 40)),
			)
		}
		prefix := ""
		if m.filterQuery != "" {
			prefix = keyStyle.Render("["+m.filterQuery+"]") + " · " + keyStyle.Render("esc") + " clear · "
		}
		return helpStyle.Render(
			"  " + prefix +
				keyStyle.Render("v") + " view · " +
				keyStyle.Render("a") + " approve · " +
				keyStyle.Render("m") + " merge · " +
				keyStyle.Render("u") + " update branch · " +
				keyStyle.Render("C") + " close · " +
				keyStyle.Render("o") + " open · " +
				keyStyle.Render("c") + " copy url · " +
				keyStyle.Render("x") + " select · " +
				keyStyle.Render("A") + " toggle all · " +
				keyStyle.Render("r") + " refresh · " +
				keyStyle.Render("q") + " quit",
		)
	}
}
