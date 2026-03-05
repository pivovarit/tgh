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
	PRChecksMsg  map[int]string
	PRReviewsMsg map[int]ReviewSummary
	ErrMsg       struct{ Err error }
	ApproveMsg   struct {
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
		args := []string{
			"search", "prs",
			"--state", "open",
			"--json", "number,title,author,state,createdAt,url,repository",
			"--limit", "100",
		}
		switch mode {
		case ModeReviewRequested:
			args = append(args, "--review-requested", "@me")
		case ModeAuthored:
			args = append(args, "--author", "@me")
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

func FetchAllCheckStatuses(prs []PR) tea.Cmd {
	return func() tea.Msg {
		if len(prs) == 0 {
			return PRChecksMsg{}
		}

		var sb strings.Builder
		sb.WriteString("query {")
		for i, pr := range prs {
			parts := strings.SplitN(pr.Repository.NameWithOwner, "/", 2)
			if len(parts) != 2 {
				continue
			}
			fmt.Fprintf(&sb,
				"\n  pr%d: repository(owner: %q, name: %q) { pullRequest(number: %d) { commits(last: 1) { nodes { commit { statusCheckRollup { state } } } } } }",
				i, parts[0], parts[1], pr.Number,
			)
		}
		sb.WriteString("\n}")

		ctx, cancel := context.WithTimeout(context.Background(), timeoutFetch)
		defer cancel()
		out, err := exec.CommandContext(ctx, "gh", "api", "graphql", "-f", "query="+sb.String()).CombinedOutput()
		if err != nil {
			return PRChecksMsg{}
		}

		var response struct {
			Data map[string]json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(out, &response); err != nil {
			return PRChecksMsg{}
		}

		type prNode struct {
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
			} `json:"pullRequest"`
		}

		result := make(PRChecksMsg)
		for i, pr := range prs {
			raw, ok := response.Data[fmt.Sprintf("pr%d", i)]
			if !ok {
				continue
			}
			var node prNode
			if err := json.Unmarshal(raw, &node); err != nil || node.PullRequest == nil {
				continue
			}
			nodes := node.PullRequest.Commits.Nodes
			if len(nodes) == 0 {
				continue
			}
			rollup := nodes[0].Commit.StatusCheckRollup
			if rollup == nil {
				result[pr.Number] = "none"
				continue
			}
			switch rollup.State {
			case "SUCCESS":
				result[pr.Number] = "success"
			case "FAILURE", "ERROR":
				result[pr.Number] = "failure"
			default:
				result[pr.Number] = "pending"
			}
		}
		return result
	}
}

func FetchAllReviewStatuses(prs []PR) tea.Cmd {
	return func() tea.Msg {
		if len(prs) == 0 {
			return PRReviewsMsg{}
		}

		var sb strings.Builder
		sb.WriteString("query {")
		for i, pr := range prs {
			parts := strings.SplitN(pr.Repository.NameWithOwner, "/", 2)
			if len(parts) != 2 {
				continue
			}
			fmt.Fprintf(&sb,
				"\n  pr%d: repository(owner: %q, name: %q) { pullRequest(number: %d) { latestReviews(first: 20) { nodes { state } } } }",
				i, parts[0], parts[1], pr.Number,
			)
		}
		sb.WriteString("\n}")

		ctx, cancel := context.WithTimeout(context.Background(), timeoutFetch)
		defer cancel()
		out, err := exec.CommandContext(ctx, "gh", "api", "graphql", "-f", "query="+sb.String()).CombinedOutput()
		if err != nil {
			return PRReviewsMsg{}
		}

		var response struct {
			Data map[string]json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(out, &response); err != nil {
			return PRReviewsMsg{}
		}

		type prNode struct {
			PullRequest *struct {
				LatestReviews struct {
					Nodes []struct {
						State string `json:"state"`
					} `json:"nodes"`
				} `json:"latestReviews"`
			} `json:"pullRequest"`
		}

		result := make(PRReviewsMsg)
		for i, pr := range prs {
			raw, ok := response.Data[fmt.Sprintf("pr%d", i)]
			if !ok {
				continue
			}
			var node prNode
			if err := json.Unmarshal(raw, &node); err != nil || node.PullRequest == nil {
				continue
			}
			var summary ReviewSummary
			for _, r := range node.PullRequest.LatestReviews.Nodes {
				switch r.State {
				case "APPROVED":
					summary.Approvals++
				case "CHANGES_REQUESTED":
					summary.ChangesRequested++
				}
			}
			if summary.Approvals > 0 || summary.ChangesRequested > 0 {
				result[pr.Number] = summary
			}
		}
		return result
	}
}

func IsArchivedError(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "archived so is read-only") || strings.Contains(s, "repository is archived")
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
