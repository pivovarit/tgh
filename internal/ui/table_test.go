package ui

import (
	"testing"

	"github.com/pivovarit/tgh/internal/gh"
)

func TestTrunc(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		max      int
		expected string
	}{
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"truncated", "hello world", 5, "hell…"},
		{"max 1", "hello", 1, "…"},
		{"max 0", "hello", 0, ""},
		{"strips leading slash", "/foo/bar", 10, "foo/bar"},
		{"unicode", "日本語テスト", 4, "日本語…"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := trunc(tt.input, tt.max); got != tt.expected {
				t.Errorf("trunc(%q, %d) = %q, want %q", tt.input, tt.max, got, tt.expected)
			}
		})
	}
}

func TestPrNum(t *testing.T) {
	if got := prNum(42); got != "#42" {
		t.Errorf("prNum(42) = %q, want %q", got, "#42")
	}
}

func TestComputeColWidths(t *testing.T) {
	prs := []gh.PR{
		{Repository: gh.Repository{NameWithOwner: "owner/repo"}, Title: "Fix bug", Author: gh.Author{Login: "user1"}},
	}
	w := computeColWidths(prs, 120)
	if w.repo < 4 {
		t.Errorf("repo width %d < minimum 4", w.repo)
	}
	if w.title < 5 {
		t.Errorf("title width %d < minimum 5", w.title)
	}
	if w.author < 6 {
		t.Errorf("author width %d < minimum 6", w.author)
	}
}

func TestComputeColWidths_Empty(t *testing.T) {
	w := computeColWidths(nil, 120)
	if w.repo < 4 || w.title < 5 || w.author < 6 {
		t.Errorf("empty PRs should still have minimum widths, got repo=%d title=%d author=%d", w.repo, w.title, w.author)
	}
}

func TestComputeColWidths_NarrowTerminal(t *testing.T) {
	prs := []gh.PR{
		{Repository: gh.Repository{NameWithOwner: "very-long-owner/very-long-repo-name"}, Title: "A very long PR title", Author: gh.Author{Login: "longusername"}},
	}
	w := computeColWidths(prs, 40)
	if w.repo < 4 {
		t.Errorf("repo width %d < minimum 4", w.repo)
	}
	if w.title < 5 {
		t.Errorf("title width %d < minimum 5", w.title)
	}
	if w.author < 6 {
		t.Errorf("author width %d < minimum 6", w.author)
	}
}

func TestStatusSymbol(t *testing.T) {
	tests := []struct {
		name  string
		ci    string
		rev   gh.ReviewSummary
		merge string
	}{
		{"success", "success", gh.ReviewSummary{}, ""},
		{"failure", "failure", gh.ReviewSummary{}, ""},
		{"pending", "pending", gh.ReviewSummary{}, ""},
		{"none", "none", gh.ReviewSummary{}, ""},
		{"unknown", "", gh.ReviewSummary{}, ""},
		{"with approvals", "success", gh.ReviewSummary{Approvals: 2}, ""},
		{"with changes requested", "success", gh.ReviewSummary{ChangesRequested: 1}, ""},
		{"behind", "success", gh.ReviewSummary{}, "BEHIND"},
		{"dirty", "success", gh.ReviewSummary{}, "DIRTY"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := statusSymbol(tt.ci, tt.rev, tt.merge)
			if got == "" {
				t.Error("statusSymbol() returned empty string")
			}
		})
	}
}

func TestBuildRows(t *testing.T) {
	prs := []gh.PR{
		{Number: 1, Repository: gh.Repository{NameWithOwner: "o/r"}, Title: "PR 1", Author: gh.Author{Login: "user"}, CreatedAt: "2025-01-01T00:00:00Z"},
		{Number: 2, Repository: gh.Repository{NameWithOwner: "o/r"}, Title: "PR 2", Author: gh.Author{Login: "user"}, CreatedAt: "2025-01-01T00:00:00Z"},
	}
	w := colWidths{repo: 10, title: 20, author: 10}
	rows := buildRows(prs, nil, nil, nil, nil, w)
	if len(rows) != 2 {
		t.Fatalf("buildRows() returned %d rows, want 2", len(rows))
	}
	if rows[0][0] != " " {
		t.Errorf("unselected row should have ' ', got %q", rows[0][0])
	}
}

func TestBuildRows_WithSelection(t *testing.T) {
	prs := []gh.PR{
		{Number: 1, Repository: gh.Repository{NameWithOwner: "o/r"}, Title: "PR 1", Author: gh.Author{Login: "user"}, CreatedAt: "2025-01-01T00:00:00Z"},
	}
	w := colWidths{repo: 10, title: 20, author: 10}
	sel := map[gh.PRKey]bool{{Num: 1, Repo: "o/r"}: true}
	rows := buildRows(prs, nil, nil, nil, sel, w)
	if rows[0][0] != "●" {
		t.Errorf("selected row should have '●', got %q", rows[0][0])
	}
}
