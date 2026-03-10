package ui

import "charm.land/lipgloss/v2"

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#38BDF8"))

	titleHintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#64748B"))

	tableStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#0369A1"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#64748B")).
			MarginTop(1)

	keyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7DD3FC")).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F87171")).
			Bold(true)

	emptyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#334155")).
			MarginLeft(2).
			MarginTop(1)

	confirmStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FCD34D")).
			Bold(true)

	confirmNameStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F8FAFC")).
				Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4ADE80")).
			Bold(true)

	warnStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FB923C")).
			Bold(true)

	dividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#0369A1"))

	panelTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#38BDF8")).
			Bold(true)

	panelLineStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CBD5E1"))

	panelSectionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#A78BFA")).
				Bold(true)

	draftRowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#52525B"))

	mergedRowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#475569"))

	closedRowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#52525B"))

	checkNoneStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#475569"))

	checkSuccessStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#4ADE80"))

	checkFailureStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F87171"))

	checkPendingStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FCD34D"))

	reviewApprovedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#4ADE80"))

	reviewChangesStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FCD34D"))

	behindStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FB923C"))

	conflictStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F87171")).
			Bold(true)

	autoMergeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#38BDF8"))
)
