package gh

import tea "charm.land/bubbletea/v2"

type Client interface {
	FetchPRs(mode PRMode, owners []string) tea.Cmd
	FetchPRDetail(number int, repo string) tea.Cmd
	FetchAllPRStatuses(prs []PR) tea.Cmd
	ApprovePR(number int, repo string) tea.Cmd
	ClosePR(number int, repo string) tea.Cmd
	MergePR(number int, repo string, strategy string) tea.Cmd
	UpdateBranch(number int, repo string) tea.Cmd
	OpenBrowser(number int, repo string) tea.Cmd
}
