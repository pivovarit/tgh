package gh

import (
	"testing"
	"time"
)

func TestCheckEntry_DisplayName(t *testing.T) {
	tests := []struct {
		name     string
		entry    CheckEntry
		expected string
	}{
		{"uses Name when set", CheckEntry{Name: "build", Context: "ci/build"}, "build"},
		{"falls back to Context", CheckEntry{Name: "", Context: "ci/build"}, "ci/build"},
		{"both empty", CheckEntry{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.entry.DisplayName(); got != tt.expected {
				t.Errorf("DisplayName() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestCheckEntry_DisplayState(t *testing.T) {
	tests := []struct {
		name     string
		entry    CheckEntry
		expected string
	}{
		{"CheckRun with conclusion", CheckEntry{TypeName: "CheckRun", Conclusion: "SUCCESS", Status: "COMPLETED"}, "SUCCESS"},
		{"CheckRun without conclusion", CheckEntry{TypeName: "CheckRun", Status: "IN_PROGRESS"}, "IN_PROGRESS"},
		{"StatusContext uses State", CheckEntry{TypeName: "StatusContext", State: "SUCCESS"}, "SUCCESS"},
		{"unknown type uses State", CheckEntry{TypeName: "Other", State: "PENDING"}, "PENDING"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.entry.DisplayState(); got != tt.expected {
				t.Errorf("DisplayState() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestPRMode_Label(t *testing.T) {
	if got := ModeReviewRequested.Label(); got != "needs review" {
		t.Errorf("ModeReviewRequested.Label() = %q, want %q", got, "needs review")
	}
	if got := ModeAuthored.Label(); got != "authored by me" {
		t.Errorf("ModeAuthored.Label() = %q, want %q", got, "authored by me")
	}
}

func TestPRMode_Next(t *testing.T) {
	if got := ModeReviewRequested.Next(); got != ModeAuthored {
		t.Errorf("ModeReviewRequested.Next() = %d, want %d", got, ModeAuthored)
	}
	if got := ModeAuthored.Next(); got != ModeReviewRequested {
		t.Errorf("ModeAuthored.Next() = %d, want %d", got, ModeReviewRequested)
	}
}

func TestIsArchivedError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"archived read-only", errStr("archived so is read-only"), true},
		{"repository is archived", errStr("Repository is archived"), true},
		{"unrelated error", errStr("permission denied"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsArchivedError(tt.err); got != tt.expected {
				t.Errorf("IsArchivedError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRelativeTime(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"invalid format", "not-a-date", "not-a-date"},
		{"just now", time.Now().UTC().Format(time.RFC3339), "now"},
		{"minutes ago", time.Now().Add(-5 * time.Minute).UTC().Format(time.RFC3339), "5m"},
		{"hours ago", time.Now().Add(-3 * time.Hour).UTC().Format(time.RFC3339), "3h"},
		{"days ago", time.Now().Add(-7 * 24 * time.Hour).UTC().Format(time.RFC3339), "7d"},
		{"months ago", time.Now().Add(-60 * 24 * time.Hour).UTC().Format(time.RFC3339), time.Now().Add(-60 * 24 * time.Hour).UTC().Format("Jan 02")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RelativeTime(tt.input); got != tt.expected {
				t.Errorf("RelativeTime(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

type errStr string

func (e errStr) Error() string { return string(e) }
