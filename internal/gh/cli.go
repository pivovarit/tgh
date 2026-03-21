package gh

import tea "charm.land/bubbletea/v2"

type CLI struct{}

func (CLI) FetchPRs(mode PRMode, owners []string) tea.Cmd { return FetchPRs(mode, owners) }
func (CLI) FetchPRDetail(number int, repo string) tea.Cmd { return FetchPRDetail(number, repo) }
func (CLI) FetchAllPRStatuses(prs []PR) tea.Cmd           { return FetchAllPRStatuses(prs) }
func (CLI) ApprovePR(number int, repo string) tea.Cmd     { return ApprovePR(number, repo) }
func (CLI) ClosePR(number int, repo string) tea.Cmd       { return ClosePR(number, repo) }
func (CLI) UpdateBranch(number int, repo string) tea.Cmd  { return UpdateBranch(number, repo) }
func (CLI) RerunChecks(number int, repo string) tea.Cmd   { return RerunChecks(number, repo) }
func (CLI) OpenBrowser(number int, repo string) tea.Cmd   { return OpenBrowser(number, repo) }
func (CLI) MergePR(number int, repo string, strategy string) tea.Cmd {
	return MergePR(number, repo, strategy)
}
func (CLI) AutoMergePR(number int, repo string, strategy string) tea.Cmd {
	return AutoMergePR(number, repo, strategy)
}
