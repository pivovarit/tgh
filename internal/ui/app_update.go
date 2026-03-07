package ui

import (
	"cmp"
	"fmt"
	"slices"

	tea "charm.land/bubbletea/v2"
	"github.com/pivovarit/tgh/internal/gh"
)

func (m App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m = m.rebuildTable(m.currentSelectedNumber())
		return m, nil

	case tea.KeyPressMsg:
		m.warnMsg = ""
		m.copiedName = ""
		if msg.String() == keyQuit || msg.String() == keyForceQuit {
			return m, tea.Quit
		}
		if m.detail.visible {
			return m.handleDetailKey(msg)
		}
		if m.op == OpConfirmApprove {
			return m.handleConfirmApproveKey(msg)
		}
		if m.op == OpConfirmClose {
			return m.handleConfirmCloseKey(msg)
		}
		if m.op == OpConfirmMerge {
			return m.handleConfirmMergeKey(msg)
		}
		if m.op == OpConfirmUpdate {
			return m.handleConfirmUpdateKey(msg)
		}
		if m.filtering {
			return m.handleFilterKey(msg)
		}
		return m.handleMainKey(msg)

	case gh.PRsMsg:
		selectedNum := m.currentSelectedNumber()
		m.prs = []gh.PR(msg)
		slices.SortStableFunc(m.prs, func(a, b gh.PR) int {
			return cmp.Compare(a.Repository.NameWithOwner, b.Repository.NameWithOwner)
		})
		m.prsByKey = indexPRs(m.prs)
		m.selected = nil
		m.checkStatus = map[gh.PRKey]string{}
		m.reviewStatus = map[gh.PRKey]gh.ReviewSummary{}
		m.mergeState = map[gh.PRKey]string{}
		m.loading = false
		m.err = nil
		m = m.rebuildTable(selectedNum)
		return m, m.client.FetchAllPRStatuses(m.prs)

	case gh.PRStatusesMsg:
		for k, v := range msg.Checks {
			m.checkStatus[k] = v
		}
		for k, v := range msg.Reviews {
			m.reviewStatus[k] = v
		}
		for k, v := range msg.MergeState {
			m.mergeState[k] = v
		}
		m = m.refreshTableRows()
		return m, nil

	case gh.ErrMsg:
		m.err = msg.Err
		m.loading = false
		m.op = OpNone
		return m, nil

	case gh.PRDetailMsg:
		if !m.detail.visible {
			return m, nil
		}
		m.detail.loading = false
		if msg.Err != nil {
			m.err = msg.Err
			m.detail.visible = false
			m.table.SetHeight(m.tableHeight())
			return m, nil
		}
		detail := msg.PR
		m.detail.pr = &detail
		m.detail.lines = buildDetailLines(detail, m.width)
		return m, nil

	case gh.ApproveMsg:
		key := gh.PRKey{Num: msg.Num, Repo: msg.Repo}
		if m.bulkPending > 0 {
			if msg.Err != nil {
				m.bulkFailed++
			} else if pr, ok := m.prsByKey[key]; ok {
				m.bulkApproved = append(m.bulkApproved, pr)
				rev := m.reviewStatus[key]
				rev.Approvals++
				m.reviewStatus[key] = rev
			}
			m.bulkPending--
			if m.bulkPending == 0 {
				m.op = OpNone
				m.warnMsg = bulkSummary("approved", m.bulkTotal, m.bulkFailed)
				m.bulkTotal = 0
				m.bulkFailed = 0
				m.selected = nil
				approved := m.bulkApproved
				m.bulkApproved = nil
				m = m.refreshTableRows()
				if len(approved) > 0 {
					return m, m.client.FetchAllPRStatuses(approved)
				}
			}
			return m, nil
		}
		m.op = OpNone
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		m.warnMsg = fmt.Sprintf("Approved #%d", msg.Num)
		rev := m.reviewStatus[key]
		rev.Approvals++
		m.reviewStatus[key] = rev
		m = m.refreshTableRows()
		if pr, ok := m.prsByKey[key]; ok {
			return m, m.client.FetchAllPRStatuses([]gh.PR{pr})
		}
		return m, nil

	case gh.CloseMsg:
		if m.bulkPending > 0 {
			if msg.Err != nil {
				m.bulkFailed++
			} else {
				m = m.removePR(msg.Num, msg.Repo)
			}
			m.bulkPending--
			if m.bulkPending == 0 {
				m.op = OpNone
				m.warnMsg = bulkSummary("closed", m.bulkTotal, m.bulkFailed)
				m.bulkTotal = 0
				m.bulkFailed = 0
				m.selected = nil
				m = m.refreshTableRows()
			}
			return m, nil
		}
		m.op = OpNone
		if msg.Err != nil {
			if gh.IsArchivedError(msg.Err) {
				m.warnMsg = "repository is archived - cannot close this PR"
			} else {
				m.err = msg.Err
			}
			return m, nil
		}
		m.warnMsg = fmt.Sprintf("Closed #%d", msg.Num)
		m = m.removePR(msg.Num, msg.Repo)
		return m, nil

	case gh.MergeMsg:
		if m.bulkPending > 0 {
			if msg.Err != nil {
				m.bulkFailed++
			} else {
				m = m.removePR(msg.Num, msg.Repo)
			}
			m.bulkPending--
			if m.bulkPending == 0 {
				m.op = OpNone
				m.warnMsg = bulkSummary("merged", m.bulkTotal, m.bulkFailed)
				m.bulkTotal = 0
				m.bulkFailed = 0
				m.selected = nil
				m = m.refreshTableRows()
			}
			return m, nil
		}
		m.op = OpNone
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		m.warnMsg = fmt.Sprintf("Merged #%d", msg.Num)
		m = m.removePR(msg.Num, msg.Repo)
		return m, nil

	case gh.UpdateBranchMsg:
		m.op = OpNone
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		m.warnMsg = fmt.Sprintf("Updated branch for #%d", msg.Num)
		if pr, ok := m.prsByKey[gh.PRKey{Num: msg.Num, Repo: msg.Repo}]; ok {
			return m, m.client.FetchAllPRStatuses([]gh.PR{pr})
		}
		return m, nil

	case clipboardMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.copiedName = msg.name
		}
		return m, nil

	case gh.NopMsg:
		return m, nil
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	cursor := m.table.Cursor()
	height := m.tableHeight()
	if cursor < m.viewportStart {
		m.viewportStart = cursor
	} else if height > 0 && cursor >= m.viewportStart+height {
		m.viewportStart = cursor - height + 1
	}
	return m, cmd
}
