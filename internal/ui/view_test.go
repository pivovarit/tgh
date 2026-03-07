package ui

import (
	"testing"

	"github.com/pivovarit/tgh/internal/gh"
)

func TestConfirmPrompt_Single(t *testing.T) {
	got := confirmPrompt("Approve", 42, "Fix bug", 0)
	if got == "" {
		t.Error("confirmPrompt should not return empty string")
	}
}

func TestConfirmPrompt_Bulk(t *testing.T) {
	got := confirmPrompt("Approve", 0, "", 5)
	if got == "" {
		t.Error("confirmPrompt should not return empty string")
	}
}

func TestProgressPrompt_Single(t *testing.T) {
	got := progressPrompt("Approving", 42, 0, 0)
	if got == "" {
		t.Error("progressPrompt should not return empty string")
	}
}

func TestProgressPrompt_Bulk(t *testing.T) {
	got := progressPrompt("Merging", 0, 5, 3)
	if got == "" {
		t.Error("progressPrompt should not return empty string")
	}
}

func TestHelpBar_Modes(t *testing.T) {
	tests := []struct {
		name string
		app  App
	}{
		{"default", func() App { a, _ := testApp(testPRs); return a }()},
		{"filtering", func() App { a, _ := testApp(testPRs); a.filtering = true; return a }()},
		{"confirming approve", func() App { a, _ := testApp(testPRs); a.op = OpConfirmApprove; return a }()},
		{"confirming close", func() App { a, _ := testApp(testPRs); a.op = OpConfirmClose; return a }()},
		{"confirming merge", func() App { a, _ := testApp(testPRs); a.op = OpConfirmMerge; return a }()},
		{"confirming update", func() App { a, _ := testApp(testPRs); a.op = OpConfirmUpdate; return a }()},
		{"approving", func() App { a, _ := testApp(testPRs); a.op = OpApproving; return a }()},
		{"closing", func() App { a, _ := testApp(testPRs); a.op = OpClosing; return a }()},
		{"merging", func() App { a, _ := testApp(testPRs); a.op = OpMerging; return a }()},
		{"updating", func() App { a, _ := testApp(testPRs); a.op = OpUpdating; return a }()},
		{"with detail", func() App { a, _ := testApp(testPRs); a.detail.visible = true; return a }()},
		{"with warn", func() App { a, _ := testApp(testPRs); a.warnMsg = "⚠ warning"; return a }()},
		{"with success warn", func() App { a, _ := testApp(testPRs); a.warnMsg = "✓ done"; return a }()},
		{"with copied", func() App { a, _ := testApp(testPRs); a.copiedName = "test-pr"; return a }()},
		{"with filter query", func() App { a, _ := testApp(testPRs); a.filterQuery = "bug"; return a }()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.app.helpBar()
			if got == "" {
				t.Error("helpBar() returned empty string")
			}
		})
	}
}

func TestView_Loading(t *testing.T) {
	app, _ := testApp(nil)
	app.loading = true
	app.width = 80
	app.height = 40
	v := app.View()
	if v.Content == "" {
		t.Error("View() should not return empty body when loading")
	}
}

func TestView_Error(t *testing.T) {
	app, _ := testApp(nil)
	app.err = errStr("something went wrong")
	app.width = 80
	app.height = 40
	v := app.View()
	if v.Content == "" {
		t.Error("View() should not return empty body on error")
	}
}

func TestView_Empty(t *testing.T) {
	app, _ := testApp(nil)
	app.width = 80
	app.height = 40
	v := app.View()
	if v.Content == "" {
		t.Error("View() should not return empty body when no PRs")
	}
}

func TestView_WithPRs(t *testing.T) {
	app, _ := testApp(testPRs)
	app.width = 120
	app.height = 40
	v := app.View()
	if v.Content == "" {
		t.Error("View() should not return empty body with PRs")
	}
}

func TestView_WithDetail(t *testing.T) {
	app, _ := testApp(testPRs)
	app.width = 120
	app.height = 40
	app.detail.visible = true
	app.detail.prNumber = 1
	app.detail.prTitle = "Fix bug"
	app.detail.loading = true
	v := app.View()
	if v.Content == "" {
		t.Error("View() should not return empty body with detail")
	}
}

func TestView_FilteredEmpty(t *testing.T) {
	app, _ := testApp(testPRs)
	app.width = 120
	app.height = 40
	app.filterQuery = "nonexistent"
	v := app.View()
	if v.Content == "" {
		t.Error("View() should not return empty body with no filter matches")
	}
}

func TestView_DraftStyling(t *testing.T) {
	prs := []gh.PR{
		{Number: 1, Title: "Draft PR", Repository: gh.Repository{NameWithOwner: "o/r"}, Author: gh.Author{Login: "u"}, CreatedAt: "2025-01-01T00:00:00Z", IsDraft: true},
		{Number: 2, Title: "Normal PR", Repository: gh.Repository{NameWithOwner: "o/r"}, Author: gh.Author{Login: "u"}, CreatedAt: "2025-01-01T00:00:00Z"},
	}
	app, _ := testApp(prs)
	app.width = 120
	app.height = 40
	v := app.View()
	if v.Content == "" {
		t.Error("View() should render draft PRs")
	}
}
