package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/pivovarit/tgh/internal/gh"
)

type mockClient struct {
	fetchPRsCalled      bool
	fetchDetailCalled   bool
	fetchStatusesCalled bool
	approveCalled       bool
	closeCalled         bool
	mergeCalled         bool
	updateBranchCalled  bool
	openBrowserCalled   bool
	lastMergeStrategy   string
	lastApproveNum      int
	lastCloseNum        int
	lastMergeNum        int
	lastUpdateBranchNum int
	lastOpenBrowserNum  int
}

func (m *mockClient) FetchPRs(mode gh.PRMode, owners []string) tea.Cmd {
	m.fetchPRsCalled = true
	return func() tea.Msg { return gh.PRsMsg{} }
}
func (m *mockClient) FetchPRDetail(number int, repo string) tea.Cmd {
	m.fetchDetailCalled = true
	return func() tea.Msg { return gh.PRDetailMsg{} }
}
func (m *mockClient) FetchAllPRStatuses(prs []gh.PR) tea.Cmd {
	m.fetchStatusesCalled = true
	return func() tea.Msg { return gh.PRStatusesMsg{} }
}
func (m *mockClient) ApprovePR(number int, repo string) tea.Cmd {
	m.approveCalled = true
	m.lastApproveNum = number
	return func() tea.Msg { return gh.ApproveMsg{Num: number, Repo: repo} }
}
func (m *mockClient) ClosePR(number int, repo string) tea.Cmd {
	m.closeCalled = true
	m.lastCloseNum = number
	return func() tea.Msg { return gh.CloseMsg{Num: number, Repo: repo} }
}
func (m *mockClient) MergePR(number int, repo string, strategy string) tea.Cmd {
	m.mergeCalled = true
	m.lastMergeNum = number
	m.lastMergeStrategy = strategy
	return func() tea.Msg { return gh.MergeMsg{Num: number, Repo: repo} }
}
func (m *mockClient) UpdateBranch(number int, repo string) tea.Cmd {
	m.updateBranchCalled = true
	m.lastUpdateBranchNum = number
	return func() tea.Msg { return gh.UpdateBranchMsg{Num: number, Repo: repo} }
}
func (m *mockClient) OpenBrowser(number int, repo string) tea.Cmd {
	m.openBrowserCalled = true
	m.lastOpenBrowserNum = number
	return func() tea.Msg { return gh.NopMsg{} }
}

func testApp(prs []gh.PR) (App, *mockClient) {
	mc := &mockClient{}
	app := newWithClient(mc, nil)
	app.width = 120
	app.height = 40
	app.prs = prs
	app.prsByKey = indexPRs(prs)
	app.loading = false
	app = app.rebuildTable(0)
	return app, mc
}

var testPRs = []gh.PR{
	{Number: 1, Title: "Fix bug", Repository: gh.Repository{NameWithOwner: "org/repo1"}, Author: gh.Author{Login: "alice"}, CreatedAt: "2025-01-01T00:00:00Z"},
	{Number: 2, Title: "Add feature", Repository: gh.Repository{NameWithOwner: "org/repo2"}, Author: gh.Author{Login: "bob"}, CreatedAt: "2025-01-02T00:00:00Z"},
	{Number: 3, Title: "Refactor code", Repository: gh.Repository{NameWithOwner: "org/repo1"}, Author: gh.Author{Login: "alice"}, CreatedAt: "2025-01-03T00:00:00Z", IsDraft: true},
}

func TestApp_Init(t *testing.T) {
	mc := &mockClient{}
	app := newWithClient(mc, nil)
	cmd := app.Init()
	if cmd == nil {
		t.Fatal("Init() should return a command")
	}
	cmd()
	if !mc.fetchPRsCalled {
		t.Error("Init() should call FetchPRs")
	}
}

func TestApp_PRsMsg(t *testing.T) {
	mc := &mockClient{}
	app := newWithClient(mc, nil)
	app.width = 120
	app.height = 40

	model, cmd := app.Update(gh.PRsMsg(testPRs))
	updated := model.(App)
	if updated.loading {
		t.Error("should not be loading after PRsMsg")
	}
	if len(updated.prs) != 3 {
		t.Errorf("expected 3 PRs, got %d", len(updated.prs))
	}
	if cmd == nil {
		t.Error("should return command to fetch statuses")
	}
}

func TestApp_PRStatusesMsg(t *testing.T) {
	app, _ := testApp(testPRs)
	key1 := gh.PRKey{Num: 1, Repo: "org/repo1"}
	key2 := gh.PRKey{Num: 2, Repo: "org/repo2"}
	msg := gh.PRStatusesMsg{
		Checks:     map[gh.PRKey]string{key1: "success", key2: "failure"},
		Reviews:    map[gh.PRKey]gh.ReviewSummary{key1: {Approvals: 1}},
		MergeState: map[gh.PRKey]string{key1: "CLEAN", key2: "BEHIND"},
	}
	model, _ := app.Update(msg)
	updated := model.(App)
	if updated.checkStatus[key1] != "success" {
		t.Errorf("check status for PR 1 = %q, want %q", updated.checkStatus[key1], "success")
	}
	if updated.mergeState[key2] != "BEHIND" {
		t.Errorf("merge state for PR 2 = %q, want %q", updated.mergeState[key2], "BEHIND")
	}
}

func TestApp_ErrMsg(t *testing.T) {
	app, _ := testApp(nil)
	app.loading = true
	model, _ := app.Update(gh.ErrMsg{Err: errStr("fail")})
	updated := model.(App)
	if updated.err == nil {
		t.Error("err should be set")
	}
	if updated.loading {
		t.Error("loading should be false")
	}
}

func TestApp_Quit(t *testing.T) {
	app, _ := testApp(testPRs)
	_, cmd := app.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	if cmd == nil {
		t.Fatal("quit should return a command")
	}
}

func TestApp_Refresh(t *testing.T) {
	app, mc := testApp(testPRs)
	model, cmd := app.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	updated := model.(App)
	if !updated.loading {
		t.Error("should be loading after refresh")
	}
	if cmd == nil {
		t.Error("should return a command")
	}
	cmd()
	if !mc.fetchPRsCalled {
		t.Error("refresh should call FetchPRs")
	}
}

func TestApp_ToggleMode(t *testing.T) {
	app, mc := testApp(testPRs)
	model, cmd := app.Update(tea.KeyPressMsg{Code: 'A', Text: "A"})
	updated := model.(App)
	if updated.prMode != gh.ModeAuthored {
		t.Errorf("mode = %d, want ModeAuthored", updated.prMode)
	}
	if !updated.loading {
		t.Error("should be loading after toggle")
	}
	if cmd == nil {
		t.Error("should return a command")
	}
	cmd()
	if !mc.fetchPRsCalled {
		t.Error("toggle should call FetchPRs")
	}
}

func TestApp_Filter(t *testing.T) {
	app, _ := testApp(testPRs)

	model, _ := app.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	updated := model.(App)
	if !updated.filtering {
		t.Error("should be in filtering mode")
	}

	model, _ = updated.Update(tea.KeyPressMsg{Code: 'b', Text: "b"})
	updated = model.(App)
	if updated.filterQuery != "b" {
		t.Errorf("filterQuery = %q, want %q", updated.filterQuery, "b")
	}

	model, _ = updated.Update(tea.KeyPressMsg{Code: 'u', Text: "u"})
	updated = model.(App)
	if updated.filterQuery != "bu" {
		t.Errorf("filterQuery = %q, want %q", updated.filterQuery, "bu")
	}

	filtered := updated.filteredPRs()
	if len(filtered) != 1 {
		t.Errorf("expected 1 filtered PR for 'bu', got %d", len(filtered))
	}

	model, _ = updated.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	updated = model.(App)
	if updated.filterQuery != "b" {
		t.Errorf("filterQuery after backspace = %q, want %q", updated.filterQuery, "b")
	}

	model, _ = updated.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	updated = model.(App)
	if updated.filtering {
		t.Error("enter should exit filtering mode")
	}
}

func TestApp_FilterByRepo(t *testing.T) {
	app, _ := testApp(testPRs)
	app.filterQuery = "repo:repo1"
	filtered := app.filteredPRs()
	if len(filtered) != 2 {
		t.Errorf("expected 2 PRs matching repo:repo1, got %d", len(filtered))
	}
}

func TestApp_FilterByAuthor(t *testing.T) {
	app, _ := testApp(testPRs)
	app.filterQuery = "author:bob"
	filtered := app.filteredPRs()
	if len(filtered) != 1 {
		t.Errorf("expected 1 PR matching author:bob, got %d", len(filtered))
	}
}

func TestApp_Select(t *testing.T) {
	app, _ := testApp(testPRs)
	model, _ := app.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	updated := model.(App)
	if len(updated.selected) != 1 {
		t.Errorf("expected 1 selected, got %d", len(updated.selected))
	}

	model, _ = updated.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	updated = model.(App)
	if len(updated.selected) != 0 {
		t.Errorf("expected 0 selected after toggle, got %d", len(updated.selected))
	}
}

func TestApp_ApproveFlow(t *testing.T) {
	app, mc := testApp(testPRs)

	model, _ := app.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	updated := model.(App)
	if updated.op != OpConfirmApprove {
		t.Errorf("op = %d, want OpConfirmApprove", updated.op)
	}

	model, cmd := updated.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	updated = model.(App)
	if updated.op != OpApproving {
		t.Errorf("op = %d, want OpApproving", updated.op)
	}
	if cmd == nil {
		t.Fatal("should return approve command")
	}
	cmd()
	if !mc.approveCalled {
		t.Error("should call ApprovePR")
	}
}

func TestApp_ApproveCancelled(t *testing.T) {
	app, _ := testApp(testPRs)

	model, _ := app.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	updated := model.(App)

	model, _ = updated.Update(tea.KeyPressMsg{Code: 'n', Text: "n"})
	updated = model.(App)
	if updated.op != OpNone {
		t.Errorf("op = %d, want OpNone after cancel", updated.op)
	}
}

func TestApp_CloseFlow(t *testing.T) {
	app, mc := testApp(testPRs)

	model, _ := app.Update(tea.KeyPressMsg{Code: 'C', Text: "C"})
	updated := model.(App)
	if updated.op != OpConfirmClose {
		t.Errorf("op = %d, want OpConfirmClose", updated.op)
	}

	model, cmd := updated.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	updated = model.(App)
	if cmd == nil {
		t.Fatal("should return close command")
	}
	cmd()
	if !mc.closeCalled {
		t.Error("should call ClosePR")
	}
}

func TestApp_CloseMsg_RemovesPR(t *testing.T) {
	app, _ := testApp(testPRs)
	model, _ := app.Update(gh.CloseMsg{Num: 1, Repo: "org/repo1"})
	updated := model.(App)
	if len(updated.prs) != 2 {
		t.Errorf("expected 2 PRs after close, got %d", len(updated.prs))
	}
}

func TestApp_CloseMsg_ArchivedError(t *testing.T) {
	app, _ := testApp(testPRs)
	model, _ := app.Update(gh.CloseMsg{Num: 1, Repo: "org/repo1", Err: errStr("repository is archived")})
	updated := model.(App)
	if updated.warnMsg == "" {
		t.Error("should set warnMsg for archived error")
	}
	if updated.err != nil {
		t.Error("should not set err for archived error")
	}
}

func TestApp_MergeFlow(t *testing.T) {
	app, mc := testApp(testPRs)

	model, _ := app.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})
	updated := model.(App)
	if updated.op != OpConfirmMerge {
		t.Errorf("op = %d, want OpConfirmMerge", updated.op)
	}

	model, cmd := updated.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	updated = model.(App)
	if cmd == nil {
		t.Fatal("should return merge command")
	}
	cmd()
	if !mc.mergeCalled {
		t.Error("should call MergePR")
	}
	if mc.lastMergeStrategy != "squash" {
		t.Errorf("merge strategy = %q, want %q", mc.lastMergeStrategy, "squash")
	}
}

func TestApp_MergeBlockedByBehind(t *testing.T) {
	app, _ := testApp(testPRs)
	app.mergeState = map[gh.PRKey]string{{Num: 1, Repo: "org/repo1"}: "BEHIND"}

	model, _ := app.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})
	updated := model.(App)
	if updated.op != OpNone {
		t.Error("should not enter merge confirm when branch is behind")
	}
	if updated.warnMsg == "" {
		t.Error("should set warnMsg when blocked")
	}
}

func TestApp_MergeBlockedByDirty(t *testing.T) {
	app, _ := testApp(testPRs)
	app.mergeState = map[gh.PRKey]string{{Num: 1, Repo: "org/repo1"}: "DIRTY"}

	model, _ := app.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})
	updated := model.(App)
	if updated.op != OpNone {
		t.Error("should not enter merge confirm when conflicts exist")
	}
}

func TestApp_MergeMsg_RemovesPR(t *testing.T) {
	app, _ := testApp(testPRs)
	model, _ := app.Update(gh.MergeMsg{Num: 1, Repo: "org/repo1"})
	updated := model.(App)
	if len(updated.prs) != 2 {
		t.Errorf("expected 2 PRs after merge, got %d", len(updated.prs))
	}
}

func TestApp_UpdateBranch(t *testing.T) {
	app, mc := testApp(testPRs)
	app.mergeState = map[gh.PRKey]string{{Num: 1, Repo: "org/repo1"}: "BEHIND"}

	model, _ := app.Update(tea.KeyPressMsg{Code: 'u', Text: "u"})
	updated := model.(App)
	if updated.op != OpConfirmUpdate {
		t.Errorf("op = %d, want OpConfirmUpdate", updated.op)
	}

	model, cmd := updated.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	updated = model.(App)
	if cmd == nil {
		t.Fatal("should return update branch command")
	}
	cmd()
	if !mc.updateBranchCalled {
		t.Error("should call UpdateBranch")
	}
}

func TestApp_UpdateBranch_NotBehind(t *testing.T) {
	app, _ := testApp(testPRs)
	app.mergeState = map[gh.PRKey]string{{Num: 1, Repo: "org/repo1"}: "CLEAN"}

	model, _ := app.Update(tea.KeyPressMsg{Code: 'u', Text: "u"})
	updated := model.(App)
	if updated.op != OpNone {
		t.Error("should not enter update confirm when branch is clean")
	}
	if updated.warnMsg == "" {
		t.Error("should set warnMsg")
	}
}

func TestApp_OpenBrowser(t *testing.T) {
	app, mc := testApp(testPRs)
	_, cmd := app.Update(tea.KeyPressMsg{Code: 'o', Text: "o"})
	if cmd == nil {
		t.Fatal("should return open browser command")
	}
	cmd()
	if !mc.openBrowserCalled {
		t.Error("should call OpenBrowser")
	}
}

func TestApp_DetailView(t *testing.T) {
	app, mc := testApp(testPRs)
	model, cmd := app.Update(tea.KeyPressMsg{Code: 'v', Text: "v"})
	updated := model.(App)
	if !updated.detail.visible {
		t.Error("detail should be visible")
	}
	if !updated.detail.loading {
		t.Error("detail should be loading")
	}
	if cmd == nil {
		t.Error("should return fetch detail command")
	}
	cmd()
	if !mc.fetchDetailCalled {
		t.Error("should call FetchPRDetail")
	}
}

func TestApp_DetailView_CloseWithEsc(t *testing.T) {
	app, _ := testApp(testPRs)
	app.detail.visible = true

	model, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	updated := model.(App)
	if updated.detail.visible {
		t.Error("detail should be closed after Esc")
	}
}

func TestApp_DetailView_CloseWithV(t *testing.T) {
	app, _ := testApp(testPRs)
	app.detail.visible = true

	model, _ := app.Update(tea.KeyPressMsg{Code: 'v', Text: "v"})
	updated := model.(App)
	if updated.detail.visible {
		t.Error("detail should be closed after v")
	}
}

func TestApp_PRDetailMsg(t *testing.T) {
	app, _ := testApp(testPRs)
	app.detail.visible = true
	app.detail.loading = true
	app.width = 120

	detail := gh.PRDetail{
		Number:    1,
		Body:      "test body",
		Additions: 10,
		Deletions: 5,
	}
	model, _ := app.Update(gh.PRDetailMsg{PR: detail})
	updated := model.(App)
	if updated.detail.loading {
		t.Error("detail should not be loading")
	}
	if updated.detail.pr == nil {
		t.Error("detail PR should be set")
	}
	if len(updated.detail.lines) == 0 {
		t.Error("detail lines should be populated")
	}
}

func TestApp_WindowSizeMsg(t *testing.T) {
	app, _ := testApp(testPRs)
	model, _ := app.Update(tea.WindowSizeMsg{Width: 200, Height: 50})
	updated := model.(App)
	if updated.width != 200 {
		t.Errorf("width = %d, want 200", updated.width)
	}
	if updated.height != 50 {
		t.Errorf("height = %d, want 50", updated.height)
	}
}

func TestApp_EscClearsSelection(t *testing.T) {
	app, _ := testApp(testPRs)
	app.selected = map[gh.PRKey]bool{{Num: 1, Repo: "org/repo1"}: true}

	model, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	updated := model.(App)
	if len(updated.selected) != 0 {
		t.Error("Esc should clear selection")
	}
}

func TestApp_EscClearsFilter(t *testing.T) {
	app, _ := testApp(testPRs)
	app.filterQuery = "something"

	model, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	updated := model.(App)
	if updated.filterQuery != "" {
		t.Errorf("filterQuery = %q, should be empty after Esc", updated.filterQuery)
	}
}

func TestApp_RemovePR(t *testing.T) {
	app, _ := testApp(testPRs)
	app = app.removePR(1, "org/repo1")
	if len(app.prs) != 2 {
		t.Errorf("expected 2 PRs after remove, got %d", len(app.prs))
	}
	for _, pr := range app.prs {
		if pr.Number == 1 && pr.Repository.NameWithOwner == "org/repo1" {
			t.Error("PR 1 should have been removed")
		}
	}
}

func TestApp_BulkApprove(t *testing.T) {
	app, mc := testApp(testPRs)
	app.selected = map[gh.PRKey]bool{
		{Num: 1, Repo: "org/repo1"}: true,
		{Num: 2, Repo: "org/repo2"}: true,
	}

	model, _ := app.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	updated := model.(App)
	if updated.op != OpConfirmApprove {
		t.Errorf("op = %d, want OpConfirmApprove", updated.op)
	}

	model, cmd := updated.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	updated = model.(App)
	if updated.op != OpApproving {
		t.Errorf("op = %d, want OpApproving", updated.op)
	}
	if updated.bulkTotal != 2 {
		t.Errorf("bulkTotal = %d, want 2", updated.bulkTotal)
	}
	if updated.bulkPending != 2 {
		t.Errorf("bulkPending = %d, want 2", updated.bulkPending)
	}
	if cmd == nil {
		t.Fatal("should return batch command")
	}
	cmd()
	if !mc.approveCalled {
		t.Error("should call ApprovePR")
	}
}

func TestApp_TableHeight(t *testing.T) {
	app, _ := testApp(testPRs)
	app.height = 40
	h := app.tableHeight()
	if h < 3 {
		t.Errorf("tableHeight() = %d, minimum is 3", h)
	}

	app.detail.visible = true
	hWithDetail := app.tableHeight()
	if hWithDetail >= h {
		t.Errorf("tableHeight with detail (%d) should be less than without (%d)", hWithDetail, h)
	}
}

func TestApp_TableHeight_SmallTerminal(t *testing.T) {
	app, _ := testApp(testPRs)
	app.height = 5
	h := app.tableHeight()
	if h != 3 {
		t.Errorf("tableHeight() = %d, want minimum 3", h)
	}
}

func TestBulkSummary(t *testing.T) {
	tests := []struct {
		verb   string
		total  int
		failed int
		want   string
	}{
		{"approved", 3, 0, "done: approved 3 PRs"},
		{"merged", 5, 2, "3 merged, 2 failed"},
	}
	for _, tt := range tests {
		got := bulkSummary(tt.verb, tt.total, tt.failed)
		if got != tt.want {
			t.Errorf("bulkSummary(%q, %d, %d) = %q, want %q", tt.verb, tt.total, tt.failed, got, tt.want)
		}
	}
}

type errStr string

func (e errStr) Error() string { return string(e) }
