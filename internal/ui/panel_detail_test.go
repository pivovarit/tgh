package ui

import (
	"testing"

	"github.com/pivovarit/tgh/internal/gh"
)

func TestBuildDetailLines_EmptyBody(t *testing.T) {
	pr := gh.PRDetail{Number: 1, Additions: 5, Deletions: 3, ChangedFiles: 2}
	lines := buildDetailLines(pr, 80)
	if len(lines) == 0 {
		t.Fatal("expected non-empty lines")
	}
	found := false
	for _, l := range lines {
		if contains(l, "(no description)") {
			found = true
		}
	}
	if !found {
		t.Error("should show '(no description)' for empty body")
	}
}

func TestBuildDetailLines_WithBody(t *testing.T) {
	pr := gh.PRDetail{Number: 1, Body: "Hello world", Additions: 1, Deletions: 0, ChangedFiles: 1}
	lines := buildDetailLines(pr, 80)
	if len(lines) == 0 {
		t.Fatal("expected non-empty lines")
	}
}

func TestBuildDetailLines_WithChecks(t *testing.T) {
	pr := gh.PRDetail{
		Number: 1,
		StatusCheckRollup: []gh.CheckEntry{
			{TypeName: "CheckRun", Name: "build", Conclusion: "SUCCESS"},
			{TypeName: "CheckRun", Name: "test", Status: "IN_PROGRESS"},
		},
	}
	lines := buildDetailLines(pr, 80)
	foundChecks := false
	for _, l := range lines {
		if contains(l, "Checks") {
			foundChecks = true
		}
	}
	if !foundChecks {
		t.Error("should have a Checks section")
	}
}

func TestBuildDetailLines_NoChecks(t *testing.T) {
	pr := gh.PRDetail{Number: 1}
	lines := buildDetailLines(pr, 80)
	found := false
	for _, l := range lines {
		if contains(l, "(no checks)") {
			found = true
		}
	}
	if !found {
		t.Error("should show '(no checks)' when no checks")
	}
}

func TestBuildDetailLines_WithReviews(t *testing.T) {
	pr := gh.PRDetail{
		Number: 1,
		LatestReviews: []gh.Review{
			{State: "APPROVED", Author: struct {
				Login string `json:"login"`
			}{Login: "reviewer1"}},
		},
	}
	lines := buildDetailLines(pr, 80)
	foundReviews := false
	for _, l := range lines {
		if contains(l, "Reviews") {
			foundReviews = true
		}
	}
	if !foundReviews {
		t.Error("should have a Reviews section")
	}
}

func TestBuildDetailLines_NoReviews(t *testing.T) {
	pr := gh.PRDetail{Number: 1}
	lines := buildDetailLines(pr, 80)
	found := false
	for _, l := range lines {
		if contains(l, "(no reviews)") {
			found = true
		}
	}
	if !found {
		t.Error("should show '(no reviews)' when no reviews")
	}
}

func TestBuildDetailLines_NarrowWidth(t *testing.T) {
	pr := gh.PRDetail{Number: 1, Body: "test", Additions: 1, Deletions: 1, ChangedFiles: 1}
	lines := buildDetailLines(pr, 10)
	if len(lines) == 0 {
		t.Fatal("should produce lines even at narrow width")
	}
}

func TestStyleCheckState(t *testing.T) {
	tests := []string{"SUCCESS", "COMPLETED", "FAILURE", "ACTION_REQUIRED", "CANCELLED", "TIMED_OUT", "IN_PROGRESS", "QUEUED"}
	for _, state := range tests {
		got := styleCheckState(state)
		if got == "" {
			t.Errorf("styleCheckState(%q) returned empty string", state)
		}
	}
}

func TestStyleReviewState(t *testing.T) {
	tests := []string{"APPROVED", "CHANGES_REQUESTED", "COMMENTED", "DISMISSED"}
	for _, state := range tests {
		got := styleReviewState(state)
		if got == "" {
			t.Errorf("styleReviewState(%q) returned empty string", state)
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && searchString(s, sub)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
