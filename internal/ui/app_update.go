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
		m.prsByNumber = indexPRs(m.prs)
		m.selected = nil
		m.checkStatus = map[int]string{}
		m.reviewStatus = map[int]gh.ReviewSummary{}
		m.loading = false
		m.err = nil
		m = m.rebuildTable(selectedNum)
		return m, tea.Batch(m.client.FetchAllCheckStatuses(m.prs), m.client.FetchAllReviewStatuses(m.prs))

	case gh.PRChecksMsg:
		m.checkStatus = map[int]string(msg)
		m = m.refreshTableRows()
		return m, nil

	case gh.PRReviewsMsg:
		m.reviewStatus = map[int]gh.ReviewSummary(msg)
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
		if m.bulkPending > 0 {
			if msg.Err != nil {
				m.bulkFailed++
			}
			m.bulkPending--
			if m.bulkPending == 0 {
				m.op = OpNone
				m.warnMsg = bulkSummary("approved", m.bulkTotal, m.bulkFailed)
				m.bulkTotal = 0
				m.bulkFailed = 0
				m.selected = nil
				m.loading = true
				return m, m.client.FetchPRs(m.prMode, m.owners)
			}
			return m, nil
		}
		m.op = OpNone
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		m.warnMsg = fmt.Sprintf("✓ Approved #%d", msg.Num)
		m.loading = true
		return m, m.client.FetchPRs(m.prMode, m.owners)

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
				m.warnMsg = "⚠ repository is archived — cannot close this PR"
			} else {
				m.err = msg.Err
			}
			return m, nil
		}
		m.warnMsg = fmt.Sprintf("✓ Closed #%d", msg.Num)
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
		m.warnMsg = fmt.Sprintf("✓ Merged #%d", msg.Num)
		m = m.removePR(msg.Num, msg.Repo)
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
