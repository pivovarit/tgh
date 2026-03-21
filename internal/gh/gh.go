package gh

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

const (
	timeoutFetch  = 15 * time.Second
	timeoutAction = 30 * time.Second
)

type Author struct {
	Login string `json:"login"`
	IsBot bool   `json:"is_bot"`
}

type Repository struct {
	Name          string `json:"name"`
	NameWithOwner string `json:"nameWithOwner"`
}

type PR struct {
	Number     int        `json:"number"`
	Title      string     `json:"title"`
	Author     Author     `json:"author"`
	State      string     `json:"state"`
	IsDraft    bool       `json:"isDraft"`
	CreatedAt  string     `json:"createdAt"`
	URL        string     `json:"url"`
	Repository Repository `json:"repository"`
}

type CheckEntry struct {
	TypeName   string `json:"__typename"`
	Name       string `json:"name"`
	Context    string `json:"context"`
	State      string `json:"state"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
}

func (c CheckEntry) DisplayName() string {
	if c.Name != "" {
		return c.Name
	}
	return c.Context
}

func (c CheckEntry) DisplayState() string {
	switch c.TypeName {
	case "CheckRun":
		if c.Conclusion != "" {
			return c.Conclusion
		}
		return c.Status
	default:
		return c.State
	}
}

type Review struct {
	Author struct {
		Login string `json:"login"`
	} `json:"author"`
	State string `json:"state"`
}

type PRDetail struct {
	Number            int          `json:"number"`
	Body              string       `json:"body"`
	StatusCheckRollup []CheckEntry `json:"statusCheckRollup"`
	LatestReviews     []Review     `json:"latestReviews"`
	Additions         int          `json:"additions"`
	Deletions         int          `json:"deletions"`
	ChangedFiles      int          `json:"changedFiles"`
}

type ReviewSummary struct {
	Approvals        int
	ChangesRequested int
}

type (
	PRsMsg      []PR
	PRDetailMsg struct {
		PR  PRDetail
		Err error
	}
	PRKey struct {
		Num  int
		Repo string
	}
	PRStatusesMsg struct {
		Checks     map[PRKey]string
		Reviews    map[PRKey]ReviewSummary
		MergeState map[PRKey]string
		AutoMerge  map[PRKey]bool
		Err        error
	}
	ErrMsg     struct{ Err error }
	ApproveMsg struct {
		Num  int
		Repo string
		Err  error
	}
	MergeMsg struct {
		Num  int
		Repo string
		Err  error
	}
	CloseMsg struct {
		Num  int
		Repo string
		Err  error
	}
	UpdateBranchMsg struct {
		Num  int
		Repo string
		Err  error
	}
	AutoMergeMsg struct {
		Num  int
		Repo string
		Err  error
	}
	RerunChecksMsg struct {
		Num  int
		Repo string
		Err  error
	}
	NopMsg struct{}
)

type PRMode int

const (
	ModeReviewRequested PRMode = iota
	ModeAuthored
)

func (m PRMode) Label() string {
	switch m {
	case ModeAuthored:
		return "authored by me"
	default:
		return "needs review"
	}
}

func (m PRMode) Next() PRMode {
	return (m + 1) % 2
}

func FetchPRs(mode PRMode, owners []string) tea.Cmd {
	return func() tea.Msg {
		if mode == ModeReviewRequested {
			return fetchReviewPRs(owners)
		}

		args := []string{
			"search", "prs",
			"--state", "open",
			"--json", "number,title,author,state,isDraft,createdAt,url,repository",
			"--limit", "100",
			"--author", "@me",
		}
		for _, o := range owners {
			args = append(args, "--owner", o)
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeoutFetch)
		defer cancel()
		out, err := exec.CommandContext(ctx, "gh", args...).CombinedOutput()
		if err != nil {
			return ErrMsg{fmt.Errorf("gh search prs: %w\n%s", err, strings.TrimSpace(string(out)))}
		}
		var prs []PR
		if err := json.Unmarshal(out, &prs); err != nil {
			return ErrMsg{fmt.Errorf("parse pr list: %w", err)}
		}
		return PRsMsg(prs)
	}
}

func fetchReviewPRs(owners []string) tea.Msg {
	const prJSON = "number,title,author,state,isDraft,createdAt,url,repository"

	type result struct {
		prs []PR
		err error
	}

	requestedCh := make(chan result, 1)
	reviewedCh := make(chan result, 1)

	ctx, cancel := context.WithTimeout(context.Background(), timeoutFetch)
	defer cancel()

	fetch := func(ch chan<- result, filter, value string) {
		args := []string{
			"search", "prs",
			"--state", "open",
			"--json", prJSON,
			"--limit", "100",
			filter, value,
		}
		for _, o := range owners {
			args = append(args, "--owner", o)
		}
		out, err := exec.CommandContext(ctx, "gh", args...).CombinedOutput()
		if err != nil {
			ch <- result{err: fmt.Errorf("gh search prs: %w\n%s", err, strings.TrimSpace(string(out)))}
			return
		}
		var prs []PR
		if err := json.Unmarshal(out, &prs); err != nil {
			ch <- result{err: fmt.Errorf("parse pr list: %w", err)}
			return
		}
		ch <- result{prs: prs}
	}

	go fetch(requestedCh, "--review-requested", "@me")
	go fetch(reviewedCh, "--reviewed-by", "@me")

	requested := <-requestedCh
	reviewed := <-reviewedCh

	if requested.err != nil && reviewed.err != nil {
		return ErrMsg{requested.err}
	}

	prs := requested.prs
	seen := make(map[PRKey]bool, len(prs))
	for _, pr := range prs {
		seen[PRKey{Num: pr.Number, Repo: pr.Repository.NameWithOwner}] = true
	}
	for _, pr := range reviewed.prs {
		if !seen[PRKey{Num: pr.Number, Repo: pr.Repository.NameWithOwner}] {
			prs = append(prs, pr)
		}
	}
	filtered, err := filterByWriteAccess(prs)
	if err != nil {
		return ErrMsg{err}
	}
	return PRsMsg(filtered)
}

func filterByWriteAccess(prs []PR) ([]PR, error) {
	if len(prs) == 0 {
		return prs, nil
	}

	repos := make(map[string]bool)
	for _, pr := range prs {
		repos[pr.Repository.NameWithOwner] = true
	}

	var sb strings.Builder
	sb.WriteString("query {")
	i := 0
	repoIndex := make(map[int]string)
	for repo := range repos {
		parts := strings.SplitN(repo, "/", 2)
		if len(parts) != 2 {
			continue
		}
		fmt.Fprintf(&sb, "\n  repo%d: repository(owner: %q, name: %q) { viewerPermission isArchived }", i, parts[0], parts[1])
		repoIndex[i] = repo
		i++
	}
	sb.WriteString("\n}")

	ctx, cancel := context.WithTimeout(context.Background(), timeoutFetch)
	defer cancel()
	out, err := exec.CommandContext(ctx, "gh", "api", "graphql", "-f", "query="+sb.String()).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("gh api graphql (write access): %w\n%s", err, strings.TrimSpace(string(out)))
	}

	var response struct {
		Data map[string]struct {
			ViewerPermission string `json:"viewerPermission"`
			IsArchived       bool   `json:"isArchived"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &response); err != nil || response.Data == nil {
		return nil, fmt.Errorf("parse write access response: %w", err)
	}

	writable := make(map[string]bool)
	for idx, repo := range repoIndex {
		if node, ok := response.Data[fmt.Sprintf("repo%d", idx)]; ok {
			if node.IsArchived {
				continue
			}
			switch node.ViewerPermission {
			case "ADMIN", "WRITE", "MAINTAIN":
				writable[repo] = true
			}
		}
	}

	filtered := make([]PR, 0, len(prs))
	for _, pr := range prs {
		if writable[pr.Repository.NameWithOwner] {
			filtered = append(filtered, pr)
		}
	}
	return filtered, nil
}

func FetchPRDetail(number int, repo string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeoutFetch)
		defer cancel()
		out, err := exec.CommandContext(ctx, "gh", "pr", "view",
			fmt.Sprintf("%d", number),
			"--repo", repo,
			"--json", "number,body,statusCheckRollup,latestReviews,additions,deletions,changedFiles",
		).CombinedOutput()
		if err != nil {
			return PRDetailMsg{Err: fmt.Errorf("gh pr view: %w\n%s", err, strings.TrimSpace(string(out)))}
		}
		var detail PRDetail
		if err := json.Unmarshal(out, &detail); err != nil {
			return PRDetailMsg{Err: fmt.Errorf("parse pr detail: %w", err)}
		}
		return PRDetailMsg{PR: detail}
	}
}

func ApprovePR(number int, repo string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeoutAction)
		defer cancel()
		out, err := exec.CommandContext(ctx, "gh", "pr", "review",
			fmt.Sprintf("%d", number),
			"--repo", repo,
			"--approve",
		).CombinedOutput()
		if err != nil {
			return ApproveMsg{Num: number, Repo: repo, Err: fmt.Errorf("gh pr review: %w\n%s", err, strings.TrimSpace(string(out)))}
		}
		return ApproveMsg{Num: number, Repo: repo}
	}
}

func MergePR(number int, repo string, strategy string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeoutAction)
		defer cancel()
		out, err := exec.CommandContext(ctx, "gh", "pr", "merge",
			fmt.Sprintf("%d", number),
			"--repo", repo,
			"--"+strategy,
		).CombinedOutput()
		if err != nil {
			return MergeMsg{Num: number, Repo: repo, Err: fmt.Errorf("gh pr merge: %w\n%s", err, strings.TrimSpace(string(out)))}
		}
		return MergeMsg{Num: number, Repo: repo}
	}
}

func AutoMergePR(number int, repo string, strategy string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeoutAction)
		defer cancel()
		out, err := exec.CommandContext(ctx, "gh", "pr", "merge",
			fmt.Sprintf("%d", number),
			"--repo", repo,
			"--"+strategy,
			"--auto",
		).CombinedOutput()
		if err != nil {
			return AutoMergeMsg{Num: number, Repo: repo, Err: fmt.Errorf("gh pr merge --auto: %w\n%s", err, strings.TrimSpace(string(out)))}
		}
		return AutoMergeMsg{Num: number, Repo: repo}
	}
}

func ClosePR(number int, repo string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeoutAction)
		defer cancel()
		out, err := exec.CommandContext(ctx, "gh", "pr", "close",
			fmt.Sprintf("%d", number),
			"--repo", repo,
		).CombinedOutput()
		if err != nil {
			return CloseMsg{Num: number, Repo: repo, Err: fmt.Errorf("gh pr close: %w\n%s", err, strings.TrimSpace(string(out)))}
		}
		return CloseMsg{Num: number, Repo: repo}
	}
}

func OpenBrowser(number int, repo string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeoutAction)
		defer cancel()
		exec.CommandContext(ctx, "gh", "pr", "view", fmt.Sprintf("%d", number), "--repo", repo, "--web").Run() //nolint:errcheck
		return NopMsg{}
	}
}

func FetchAllPRStatuses(prs []PR) tea.Cmd {
	cmds := make([]tea.Cmd, len(prs))
	for i, pr := range prs {
		cmds[i] = fetchPRStatus(pr)
	}
	return tea.Batch(cmds...)
}

func fetchPRStatus(pr PR) tea.Cmd {
	return func() tea.Msg {
		parts := strings.SplitN(pr.Repository.NameWithOwner, "/", 2)
		if len(parts) != 2 {
			return PRStatusesMsg{}
		}

		query := fmt.Sprintf(
			`query { repository(owner: %q, name: %q) { pullRequest(number: %d) {`+
				` commits(last: 1) { nodes { commit { statusCheckRollup { state } } } }`+
				` latestReviews(first: 20) { nodes { state } }`+
				` mergeStateStatus`+
				` autoMergeRequest { enabledAt }`+
				` } } }`,
			parts[0], parts[1], pr.Number,
		)

		ctx, cancel := context.WithTimeout(context.Background(), timeoutFetch)
		defer cancel()
		out, err := exec.CommandContext(ctx, "gh", "api", "graphql", "-f", "query="+query).Output()
		if err != nil {
			return PRStatusesMsg{Err: fmt.Errorf("gh api graphql (#%d): %w", pr.Number, err)}
		}

		var resp struct {
			Data struct {
				Repository struct {
					PullRequest *struct {
						Commits struct {
							Nodes []struct {
								Commit struct {
									StatusCheckRollup *struct {
										State string `json:"state"`
									} `json:"statusCheckRollup"`
								} `json:"commit"`
							} `json:"nodes"`
						} `json:"commits"`
						LatestReviews struct {
							Nodes []struct {
								State string `json:"state"`
							} `json:"nodes"`
						} `json:"latestReviews"`
						MergeStateStatus string `json:"mergeStateStatus"`
						AutoMergeRequest *struct {
							EnabledAt string `json:"enabledAt"`
						} `json:"autoMergeRequest"`
					} `json:"pullRequest"`
				} `json:"repository"`
			} `json:"data"`
		}
		if err := json.Unmarshal(out, &resp); err != nil {
			return PRStatusesMsg{Err: fmt.Errorf("parse pr status (#%d): %w", pr.Number, err)}
		}

		key := PRKey{Num: pr.Number, Repo: pr.Repository.NameWithOwner}
		result := PRStatusesMsg{
			Checks:     make(map[PRKey]string),
			Reviews:    make(map[PRKey]ReviewSummary),
			MergeState: make(map[PRKey]string),
			AutoMerge:  make(map[PRKey]bool),
		}

		p := resp.Data.Repository.PullRequest
		if p == nil {
			return result
		}

		if nodes := p.Commits.Nodes; len(nodes) > 0 {
			if rollup := nodes[0].Commit.StatusCheckRollup; rollup == nil {
				result.Checks[key] = "none"
			} else {
				switch rollup.State {
				case "SUCCESS":
					result.Checks[key] = "success"
				case "FAILURE", "ERROR":
					result.Checks[key] = "failure"
				default:
					result.Checks[key] = "pending"
				}
			}
		}

		var summary ReviewSummary
		for _, r := range p.LatestReviews.Nodes {
			switch r.State {
			case "APPROVED":
				summary.Approvals++
			case "CHANGES_REQUESTED":
				summary.ChangesRequested++
			}
		}
		if summary.Approvals > 0 || summary.ChangesRequested > 0 {
			result.Reviews[key] = summary
		}

		result.MergeState[key] = p.MergeStateStatus
		result.AutoMerge[key] = p.AutoMergeRequest != nil
		return result
	}
}

func UpdateBranch(number int, repo string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeoutAction)
		defer cancel()
		out, err := exec.CommandContext(ctx, "gh", "pr", "update-branch",
			fmt.Sprintf("%d", number),
			"--repo", repo,
		).CombinedOutput()
		if err != nil {
			return UpdateBranchMsg{Num: number, Repo: repo, Err: fmt.Errorf("gh pr update-branch: %w\n%s", err, strings.TrimSpace(string(out)))}
		}
		return UpdateBranchMsg{Num: number, Repo: repo}
	}
}

func RerunChecks(number int, repo string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeoutAction)
		defer cancel()

		shaOut, err := exec.CommandContext(ctx, "gh", "api",
			fmt.Sprintf("repos/%s/pulls/%d", repo, number),
			"--jq", ".head.sha",
		).Output()
		if err != nil {
			return RerunChecksMsg{Num: number, Repo: repo, Err: fmt.Errorf("get PR head SHA: %w", err)}
		}
		sha := strings.TrimSpace(string(shaOut))

		runsOut, err := exec.CommandContext(ctx, "gh", "api",
			fmt.Sprintf("repos/%s/actions/runs?head_sha=%s&per_page=20", repo, sha),
			"--jq", `.workflow_runs[] | select(.conclusion == "failure" or .conclusion == "startup_failure") | .id`,
		).Output()
		if err != nil {
			return RerunChecksMsg{Num: number, Repo: repo, Err: fmt.Errorf("list runs: %w", err)}
		}

		runIDs := strings.Fields(strings.TrimSpace(string(runsOut)))
		if len(runIDs) == 0 {
			return RerunChecksMsg{Num: number, Repo: repo, Err: fmt.Errorf("no failed runs to rerun")}
		}

		for _, runID := range runIDs {
			out, err := exec.CommandContext(ctx, "gh", "run", "rerun", runID, "--failed", "--repo", repo).CombinedOutput()
			if err != nil {
				return RerunChecksMsg{Num: number, Repo: repo, Err: fmt.Errorf("rerun %s: %w\n%s", runID, err, strings.TrimSpace(string(out)))}
			}
		}
		return RerunChecksMsg{Num: number, Repo: repo}
	}
}

func RelativeTime(isoTime string) string {
	t, err := time.Parse(time.RFC3339, isoTime)
	if err != nil {
		return isoTime
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	default:
		return t.Format("Jan 02")
	}
}
