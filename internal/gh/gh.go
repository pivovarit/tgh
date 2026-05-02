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

func ghGraphQL(ctx context.Context, query string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "gh", "api", "graphql", "-f", "query="+query)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("graphql: %w\n%s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("graphql: %w", err)
	}
	return out, nil
}

func prNodeID(ctx context.Context, number int, repo string) (string, error) {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid repo: %s", repo)
	}
	query := fmt.Sprintf(`{ repository(owner: %q, name: %q) { pullRequest(number: %d) { id } } }`,
		parts[0], parts[1], number)
	out, err := ghGraphQL(ctx, query)
	if err != nil {
		return "", err
	}
	var resp struct {
		Data struct {
			Repository struct {
				PullRequest struct {
					ID string `json:"id"`
				} `json:"pullRequest"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return "", fmt.Errorf("parse node ID: %w", err)
	}
	if resp.Data.Repository.PullRequest.ID == "" {
		return "", fmt.Errorf("PR #%d not found in %s", number, repo)
	}
	return resp.Data.Repository.PullRequest.ID, nil
}

func mergeMethod(strategy string) string {
	switch strategy {
	case "squash":
		return "SQUASH"
	case "rebase":
		return "REBASE"
	default:
		return "MERGE"
	}
}

func ownerQualifiers(ctx context.Context, owners []string) string {
	if len(owners) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("{")
	for i, owner := range owners {
		fmt.Fprintf(&sb, " owner%d: repositoryOwner(login: %q) { __typename }", i, owner)
	}
	sb.WriteString(" }")
	out, err := ghGraphQL(ctx, sb.String())
	if err != nil {
		var result strings.Builder
		for _, o := range owners {
			result.WriteString(" user:" + o)
		}
		return result.String()
	}
	var resp struct {
		Data map[string]struct {
			TypeName string `json:"__typename"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		var result strings.Builder
		for _, o := range owners {
			result.WriteString(" user:" + o)
		}
		return result.String()
	}
	var result strings.Builder
	for i, owner := range owners {
		key := fmt.Sprintf("owner%d", i)
		if node, ok := resp.Data[key]; ok && node.TypeName == "Organization" {
			result.WriteString(" org:" + owner)
		} else {
			result.WriteString(" user:" + owner)
		}
	}
	return result.String()
}

func searchPRsGraphQL(ctx context.Context, searchQuery string) ([]PR, error) {
	query := fmt.Sprintf(`{
  search(query: %q, type: ISSUE, first: 100) {
    nodes {
      ... on PullRequest {
        number
        title
        author { login }
        state
        isDraft
        createdAt
        url
        repository { name nameWithOwner }
      }
    }
  }
}`, searchQuery)

	out, err := ghGraphQL(ctx, query)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data struct {
			Search struct {
				Nodes []struct {
					Number     int    `json:"number"`
					Title      string `json:"title"`
					Author     struct {
						Login string `json:"login"`
					} `json:"author"`
					State      string `json:"state"`
					IsDraft    bool   `json:"isDraft"`
					CreatedAt  string `json:"createdAt"`
					URL        string `json:"url"`
					Repository struct {
						Name          string `json:"name"`
						NameWithOwner string `json:"nameWithOwner"`
					} `json:"repository"`
				} `json:"nodes"`
			} `json:"search"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parse search results: %w", err)
	}

	prs := make([]PR, 0, len(resp.Data.Search.Nodes))
	for _, n := range resp.Data.Search.Nodes {
		if n.Number == 0 {
			continue
		}
		prs = append(prs, PR{
			Number:     n.Number,
			Title:      n.Title,
			Author:     Author{Login: n.Author.Login},
			State:      n.State,
			IsDraft:    n.IsDraft,
			CreatedAt:  n.CreatedAt,
			URL:        n.URL,
			Repository: Repository{Name: n.Repository.Name, NameWithOwner: n.Repository.NameWithOwner},
		})
	}
	return prs, nil
}

func FetchPRs(mode PRMode, owners []string) tea.Cmd {
	return func() tea.Msg {
		if mode == ModeReviewRequested {
			return fetchReviewPRs(owners)
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeoutFetch)
		defer cancel()

		searchQuery := "is:pr is:open author:@me" + ownerQualifiers(ctx, owners)
		prs, err := searchPRsGraphQL(ctx, searchQuery)
		if err != nil {
			return ErrMsg{err}
		}
		return PRsMsg(prs)
	}
}

func fetchReviewPRs(owners []string) tea.Msg {
	type result struct {
		prs []PR
		err error
	}

	requestedCh := make(chan result, 1)
	reviewedCh := make(chan result, 1)

	ctx, cancel := context.WithTimeout(context.Background(), timeoutFetch)
	defer cancel()

	qualifiers := ownerQualifiers(ctx, owners)

	fetch := func(ch chan<- result, filter string) {
		prs, err := searchPRsGraphQL(ctx, "is:pr is:open "+filter+qualifiers)
		ch <- result{prs: prs, err: err}
	}

	go fetch(requestedCh, "review-requested:@me")
	go fetch(reviewedCh, "reviewed-by:@me")

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
	out, err := ghGraphQL(ctx, sb.String())
	if err != nil {
		return nil, err
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
		parts := strings.SplitN(repo, "/", 2)
		if len(parts) != 2 {
			return PRDetailMsg{Err: fmt.Errorf("invalid repo: %s", repo)}
		}
		query := fmt.Sprintf(`{
  repository(owner: %q, name: %q) {
    pullRequest(number: %d) {
      number
      body
      additions
      deletions
      changedFiles
      commits(last: 1) {
        nodes {
          commit {
            statusCheckRollup {
              contexts(first: 100) {
                nodes {
                  __typename
                  ... on CheckRun { name status conclusion }
                  ... on StatusContext { context state }
                }
              }
            }
          }
        }
      }
      latestReviews(first: 100) {
        nodes {
          author { login }
          state
        }
      }
    }
  }
}`, parts[0], parts[1], number)

		ctx, cancel := context.WithTimeout(context.Background(), timeoutFetch)
		defer cancel()
		out, err := ghGraphQL(ctx, query)
		if err != nil {
			return PRDetailMsg{Err: err}
		}

		var resp struct {
			Data struct {
				Repository struct {
					PullRequest *struct {
						Number       int    `json:"number"`
						Body         string `json:"body"`
						Additions    int    `json:"additions"`
						Deletions    int    `json:"deletions"`
						ChangedFiles int    `json:"changedFiles"`
						Commits      struct {
							Nodes []struct {
								Commit struct {
									StatusCheckRollup *struct {
										Contexts struct {
											Nodes []struct {
												TypeName   string `json:"__typename"`
												Name       string `json:"name"`
												Status     string `json:"status"`
												Conclusion string `json:"conclusion"`
												Context    string `json:"context"`
												State      string `json:"state"`
											} `json:"nodes"`
										} `json:"contexts"`
									} `json:"statusCheckRollup"`
								} `json:"commit"`
							} `json:"nodes"`
						} `json:"commits"`
						LatestReviews struct {
							Nodes []struct {
								Author struct {
									Login string `json:"login"`
								} `json:"author"`
								State string `json:"state"`
							} `json:"nodes"`
						} `json:"latestReviews"`
					} `json:"pullRequest"`
				} `json:"repository"`
			} `json:"data"`
		}
		if err := json.Unmarshal(out, &resp); err != nil {
			return PRDetailMsg{Err: fmt.Errorf("parse pr detail: %w", err)}
		}

		p := resp.Data.Repository.PullRequest
		if p == nil {
			return PRDetailMsg{Err: fmt.Errorf("PR #%d not found in %s", number, repo)}
		}

		detail := PRDetail{
			Number:       p.Number,
			Body:         p.Body,
			Additions:    p.Additions,
			Deletions:    p.Deletions,
			ChangedFiles: p.ChangedFiles,
		}

		if len(p.Commits.Nodes) > 0 {
			if rollup := p.Commits.Nodes[0].Commit.StatusCheckRollup; rollup != nil {
				for _, c := range rollup.Contexts.Nodes {
					detail.StatusCheckRollup = append(detail.StatusCheckRollup, CheckEntry{
						TypeName:   c.TypeName,
						Name:       c.Name,
						Status:     c.Status,
						Conclusion: c.Conclusion,
						Context:    c.Context,
						State:      c.State,
					})
				}
			}
		}

		for _, r := range p.LatestReviews.Nodes {
			detail.LatestReviews = append(detail.LatestReviews, Review{
				Author: struct {
					Login string `json:"login"`
				}{Login: r.Author.Login},
				State: r.State,
			})
		}

		return PRDetailMsg{PR: detail}
	}
}

func ApprovePR(number int, repo string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeoutAction)
		defer cancel()
		nodeID, err := prNodeID(ctx, number, repo)
		if err != nil {
			return ApproveMsg{Num: number, Repo: repo, Err: err}
		}
		mutation := fmt.Sprintf(`mutation {
  addPullRequestReview(input: {pullRequestId: %q, event: APPROVE}) {
    clientMutationId
  }
}`, nodeID)
		if _, err := ghGraphQL(ctx, mutation); err != nil {
			return ApproveMsg{Num: number, Repo: repo, Err: err}
		}
		return ApproveMsg{Num: number, Repo: repo}
	}
}

func MergePR(number int, repo string, strategy string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeoutAction)
		defer cancel()
		nodeID, err := prNodeID(ctx, number, repo)
		if err != nil {
			return MergeMsg{Num: number, Repo: repo, Err: err}
		}
		mutation := fmt.Sprintf(`mutation {
  mergePullRequest(input: {pullRequestId: %q, mergeMethod: %s}) {
    clientMutationId
  }
}`, nodeID, mergeMethod(strategy))
		if _, err := ghGraphQL(ctx, mutation); err != nil {
			return MergeMsg{Num: number, Repo: repo, Err: err}
		}
		return MergeMsg{Num: number, Repo: repo}
	}
}

func AutoMergePR(number int, repo string, strategy string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeoutAction)
		defer cancel()
		nodeID, err := prNodeID(ctx, number, repo)
		if err != nil {
			return AutoMergeMsg{Num: number, Repo: repo, Err: err}
		}
		method := mergeMethod(strategy)
		mutation := fmt.Sprintf(`mutation {
  enablePullRequestAutoMerge(input: {pullRequestId: %q, mergeMethod: %s}) {
    clientMutationId
  }
}`, nodeID, method)
		if _, err := ghGraphQL(ctx, mutation); err != nil {
			fallback := fmt.Sprintf(`mutation {
  mergePullRequest(input: {pullRequestId: %q, mergeMethod: %s}) {
    clientMutationId
  }
}`, nodeID, method)
			if _, err2 := ghGraphQL(ctx, fallback); err2 != nil {
				return AutoMergeMsg{Num: number, Repo: repo, Err: err}
			}
		}
		return AutoMergeMsg{Num: number, Repo: repo}
	}
}

func ClosePR(number int, repo string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeoutAction)
		defer cancel()
		nodeID, err := prNodeID(ctx, number, repo)
		if err != nil {
			return CloseMsg{Num: number, Repo: repo, Err: err}
		}
		mutation := fmt.Sprintf(`mutation {
  closePullRequest(input: {pullRequestId: %q}) {
    clientMutationId
  }
}`, nodeID)
		if _, err := ghGraphQL(ctx, mutation); err != nil {
			return CloseMsg{Num: number, Repo: repo, Err: err}
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
	const maxParallel = 20
	cmds := make([]tea.Cmd, len(prs))
	sem := make(chan struct{}, maxParallel)
	for i, pr := range prs {
		cmds[i] = fetchPRStatusWithSem(pr, sem)
	}
	return tea.Batch(cmds...)
}

func fetchPRStatusWithSem(pr PR, sem chan struct{}) tea.Cmd {
	return func() tea.Msg {
		sem <- struct{}{}
		defer func() { <-sem }()
		return fetchPRStatus(pr)()
	}
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
		out, err := ghGraphQL(ctx, query)
		if err != nil {
			return PRStatusesMsg{Err: fmt.Errorf("graphql (#%d): %w", pr.Number, err)}
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
		nodeID, err := prNodeID(ctx, number, repo)
		if err != nil {
			return UpdateBranchMsg{Num: number, Repo: repo, Err: err}
		}
		mutation := fmt.Sprintf(`mutation {
  updatePullRequestBranch(input: {pullRequestId: %q}) {
    pullRequest { id }
  }
}`, nodeID)
		if _, err := ghGraphQL(ctx, mutation); err != nil {
			return UpdateBranchMsg{Num: number, Repo: repo, Err: err}
		}
		return UpdateBranchMsg{Num: number, Repo: repo}
	}
}

func RerunChecks(number int, repo string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeoutAction)
		defer cancel()

		parts := strings.SplitN(repo, "/", 2)
		if len(parts) != 2 {
			return RerunChecksMsg{Num: number, Repo: repo, Err: fmt.Errorf("invalid repo: %s", repo)}
		}

		query := fmt.Sprintf(`{ repository(owner: %q, name: %q) { pullRequest(number: %d) { headRefOid } } }`,
			parts[0], parts[1], number)
		out, err := ghGraphQL(ctx, query)
		if err != nil {
			return RerunChecksMsg{Num: number, Repo: repo, Err: fmt.Errorf("get PR head SHA: %w", err)}
		}
		var shaResp struct {
			Data struct {
				Repository struct {
					PullRequest struct {
						HeadRefOid string `json:"headRefOid"`
					} `json:"pullRequest"`
				} `json:"repository"`
			} `json:"data"`
		}
		if err := json.Unmarshal(out, &shaResp); err != nil {
			return RerunChecksMsg{Num: number, Repo: repo, Err: fmt.Errorf("parse head SHA: %w", err)}
		}
		sha := shaResp.Data.Repository.PullRequest.HeadRefOid

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
			out, err := exec.CommandContext(ctx, "gh", "api", "-X", "POST",
				fmt.Sprintf("repos/%s/actions/runs/%s/rerun-failed-jobs", repo, runID),
			).CombinedOutput()
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
