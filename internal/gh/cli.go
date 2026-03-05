package gh

import tea "charm.land/bubbletea/v2"

type CLI struct{}

func (CLI) FetchPRs(mode PRMode, owners []string) tea.Cmd { return FetchPRs(mode, owners) }
func (CLI) FetchPRDetail(number int, repo string) tea.Cmd       { return FetchPRDetail(number, repo) }
func (CLI) FetchAllCheckStatuses(prs []PR) tea.Cmd             { return FetchAllCheckStatuses(prs) }
func (CLI) FetchAllReviewStatuses(prs []PR) tea.Cmd            { return FetchAllReviewStatuses(prs) }
func (CLI) ApprovePR(number int, repo string) tea.Cmd            { return ApprovePR(number, repo) }
func (CLI) ClosePR(number int, repo string) tea.Cmd              { return ClosePR(number, repo) }
func (CLI) MergePR(number int, repo string, strategy string) tea.Cmd {
	return MergePR(number, repo, strategy)
}
func (CLI) OpenBrowser(number int, repo string) tea.Cmd { return OpenBrowser(number, repo) }
