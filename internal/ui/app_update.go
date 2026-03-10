package ui

import (
	"cmp"
	"fmt"
	"slices"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/pivovarit/tgh/internal/gh"
)

const statusRecheckDelay = 5 * time.Second

type recheckMsg struct {
	prs []gh.PR
}

func (m App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m = m.rebuildTable(m.currentSelectedKey())
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
		if m.op == OpConfirmAutoMerge {
			return m.handleConfirmAutoMergeKey(msg)
		}
		if m.filtering {
			if isFilterKey(msg) {
				return m.handleFilterKey(msg)
			}
			m.filtering = false
		}
		return m.handleMainKey(msg)

	case gh.PRsMsg:
		selectedKey := m.currentSelectedKey()
		m.prs = []gh.PR(msg)
		slices.SortStableFunc(m.prs, func(a, b gh.PR) int {
			return cmp.Compare(a.Repository.NameWithOwner, b.Repository.NameWithOwner)
		})
		m.prsByKey = indexPRs(m.prs)
		if m.selected != nil {
			for k := range m.selected {
				if _, exists := m.prsByKey[k]; !exists {
					delete(m.selected, k)
				}
			}
			if len(m.selected) == 0 {
				m.selected = nil
			}
		}
		m.checkStatus = map[gh.PRKey]string{}
		m.reviewStatus = map[gh.PRKey]gh.ReviewSummary{}
		m.mergeState = map[gh.PRKey]string{}
		m.autoMergeStatus = map[gh.PRKey]bool{}
		m.loading = false
		m.err = nil
		m = m.rebuildTable(selectedKey)
		return m, m.client.FetchAllPRStatuses(m.prs)

	case gh.PRStatusesMsg:
		if msg.Err != nil {
			m.warnMsg = fmt.Sprintf("⚠ %s", msg.Err)
			return m, nil
		}
		for k, v := range msg.Checks {
			m.checkStatus[k] = v
		}
		for k, v := range msg.Reviews {
			m.reviewStatus[k] = v
		}
		for k, v := range msg.MergeState {
			m.mergeState[k] = v
		}
		for k, v := range msg.AutoMerge {
			m.autoMergeStatus[k] = v
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
			m.err = msg.Err
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
				m = m.invalidateRepoStatuses(msg.Repo)
				m = m.refreshTableRows()
				repoPRs := m.prsInRepo(msg.Repo)
				if len(repoPRs) > 0 {
					return m, tea.Every(statusRecheckDelay, func(time.Time) tea.Msg {
						return recheckMsg{prs: repoPRs}
					})
				}
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
		m = m.invalidateRepoStatuses(msg.Repo)
		repoPRs := m.prsInRepo(msg.Repo)
		if len(repoPRs) > 0 {
			return m, tea.Every(statusRecheckDelay, func(time.Time) tea.Msg {
				return recheckMsg{prs: repoPRs}
			})
		}
		return m, nil

	case gh.AutoMergeMsg:
		if m.bulkPending > 0 {
			if msg.Err != nil {
				m.bulkFailed++
			} else {
				key := gh.PRKey{Num: msg.Num, Repo: msg.Repo}
				m.autoMergeStatus[key] = true
			}
			m.bulkPending--
			if m.bulkPending == 0 {
				m.op = OpNone
				m.warnMsg = bulkSummary("auto-merge enabled", m.bulkTotal, m.bulkFailed)
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
		key := gh.PRKey{Num: msg.Num, Repo: msg.Repo}
		m.warnMsg = fmt.Sprintf("Auto-merge enabled for #%d", msg.Num)
		m.autoMergeStatus[key] = true
		m = m.refreshTableRows()
		return m, nil

	case gh.UpdateBranchMsg:
		m.op = OpNone
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		key := gh.PRKey{Num: msg.Num, Repo: msg.Repo}
		m.warnMsg = fmt.Sprintf("Updated branch for #%d", msg.Num)
		m.checkStatus[key] = "pending"
		m.mergeState[key] = ""
		m = m.refreshTableRows()
		if pr, ok := m.prsByKey[key]; ok {
			return m, tea.Every(statusRecheckDelay, func(time.Time) tea.Msg {
				return recheckMsg{prs: []gh.PR{pr}}
			})
		}
		return m, nil

	case recheckMsg:
		return m, m.client.FetchAllPRStatuses(msg.prs)

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

	return m, nil
}
