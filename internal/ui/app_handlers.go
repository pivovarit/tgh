package ui

import (
	"strconv"

	tea "charm.land/bubbletea/v2"
	"github.com/pivovarit/tgh/internal/gh"
)

func (m App) handleMainKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.Text {
	case keyRefresh:
		m.selected = nil
		m.loading = true
		m.err = nil
		return m, m.client.FetchPRs(m.prMode, m.owners)
	case keyToggleAll:
		m.selected = nil
		m.prMode = m.prMode.Next()
		m.loading = true
		m.err = nil
		return m, m.client.FetchPRs(m.prMode, m.owners)
	case keyFilter:
		m.filtering = true
		return m, nil
	case keySelect:
		m = m.toggleSelect()
		return m, nil
	case keyDetail:
		if pr := m.currentPR(); pr != nil {
			m.detail.visible = true
			m.detail.prNumber = pr.Number
			m.detail.prTitle = pr.Title
			m.detail.pr = nil
			m.detail.lines = nil
			m.detail.scroll = scrollState{}
			m.detail.loading = true
			m.table.SetHeight(m.tableHeight())
			return m, m.client.FetchPRDetail(pr.Number, pr.Repository.NameWithOwner)
		}
	case keyApprove:
		if sel := m.selectedPRs(); len(sel) > 0 {
			m.op = OpConfirmApprove
			m.confirmNum = 0
			m.confirmTitle = ""
			m.confirmRepo = ""
		} else if pr := m.currentPR(); pr != nil {
			m.op = OpConfirmApprove
			m.confirmNum = pr.Number
			m.confirmTitle = pr.Title
			m.confirmRepo = pr.Repository.NameWithOwner
		}
		return m, nil
	case keyClose:
		if sel := m.selectedPRs(); len(sel) > 0 {
			m.op = OpConfirmClose
			m.confirmNum = 0
			m.confirmTitle = ""
			m.confirmRepo = ""
		} else if pr := m.currentPR(); pr != nil {
			m.op = OpConfirmClose
			m.confirmNum = pr.Number
			m.confirmTitle = pr.Title
			m.confirmRepo = pr.Repository.NameWithOwner
		}
		return m, nil
	case keyMerge:
		if sel := m.selectedPRs(); len(sel) > 0 {
			for _, pr := range sel {
				switch m.mergeState[keyFor(pr)] {
				case "BEHIND":
					m.warnMsg = "cannot merge: some PRs are not up to date with the base branch"
					return m, nil
				case "DIRTY":
					m.warnMsg = "cannot merge: some PRs have merge conflicts"
					return m, nil
				}
			}
			m.op = OpConfirmMerge
			m.confirmNum = 0
			m.confirmTitle = ""
			m.confirmRepo = ""
		} else if pr := m.currentPR(); pr != nil {
			switch m.mergeState[keyFor(*pr)] {
			case "BEHIND":
				m.warnMsg = "cannot merge: branch is not up to date with the base branch"
				return m, nil
			case "DIRTY":
				m.warnMsg = "cannot merge: branch has merge conflicts"
				return m, nil
			}
			m.op = OpConfirmMerge
			m.confirmNum = pr.Number
			m.confirmTitle = pr.Title
			m.confirmRepo = pr.Repository.NameWithOwner
		}
		return m, nil
	case keyAutoMerge:
		if sel := m.selectedPRs(); len(sel) > 0 {
			m.op = OpConfirmAutoMerge
			m.confirmNum = 0
			m.confirmTitle = ""
			m.confirmRepo = ""
		} else if pr := m.currentPR(); pr != nil {
			m.op = OpConfirmAutoMerge
			m.confirmNum = pr.Number
			m.confirmTitle = pr.Title
			m.confirmRepo = pr.Repository.NameWithOwner
		}
		return m, nil
	case keyUpdate:
		if pr := m.currentPR(); pr != nil {
			if m.mergeState[keyFor(*pr)] != "BEHIND" {
				m.warnMsg = "branch is already up to date with the base branch"
				return m, nil
			}
			m.op = OpConfirmUpdate
			m.confirmNum = pr.Number
			m.confirmTitle = pr.Title
			m.confirmRepo = pr.Repository.NameWithOwner
		}
		return m, nil
	case keyRerun:
		if pr := m.currentPR(); pr != nil {
			if m.checkStatus[keyFor(*pr)] != "failure" {
				m.warnMsg = "no failed checks to rerun"
				return m, nil
			}
			m.op = OpConfirmRerun
			m.confirmNum = pr.Number
			m.confirmTitle = pr.Title
			m.confirmRepo = pr.Repository.NameWithOwner
		}
		return m, nil
	case keyBrowser:
		if pr := m.currentPR(); pr != nil {
			return m, m.client.OpenBrowser(pr.Number, pr.Repository.NameWithOwner)
		}
	case keyCopy:
		if pr := m.currentPR(); pr != nil {
			return m, copyToClipboard(pr.Title, pr.URL)
		}
	default:
		if msg.Code == tea.KeyEsc && m.op == OpNone {
			if len(m.selected) > 0 {
				m.selected = nil
				m = m.refreshTableRows()
				return m, nil
			} else if m.filterQuery != "" {
				selectedKey := m.currentSelectedKey()
				m.filterQuery = ""
				m = m.rebuildTable(selectedKey)
			} else if m.err != nil {
				m.err = nil
			}
		}
	}

	n := len(m.filteredPRs())
	height := m.visibleRows()
	switch msg.Code {
	case tea.KeyUp:
		m = m.moveCursor(-1, n, height)
	case tea.KeyDown:
		m = m.moveCursor(1, n, height)
	case tea.KeyPgUp:
		m = m.moveCursor(-height, n, height)
	case tea.KeyPgDown:
		m = m.moveCursor(height, n, height)
	case tea.KeyHome:
		m = m.moveCursor(-n, n, height)
	case tea.KeyEnd:
		m = m.moveCursor(n, n, height)
	default:
		switch msg.Text {
		case keyVimDown:
			m = m.moveCursor(1, n, height)
		case keyVimUp:
			m = m.moveCursor(-1, n, height)
		case "g":
			m = m.moveCursor(-n, n, height)
		case keyScrollBottom:
			m = m.moveCursor(n, n, height)
		}
	}
	return m, nil
}

func (m App) handleDetailKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.Code {
	case tea.KeyEsc:
		m = m.closeDetail()
		return m, nil
	case tea.KeyUp:
		m.detail.scroll = m.detail.scroll.up()
	case tea.KeyDown:
		m.detail.scroll = m.detail.scroll.down(len(m.detail.lines), detailPanelHeight-2)
	}
	switch msg.Text {
	case keyDetail:
		m = m.closeDetail()
		return m, nil
	case "g":
		m.detail.scroll = m.detail.scroll.top()
	case keyScrollBottom:
		m.detail.scroll = m.detail.scroll.bottom(len(m.detail.lines), detailPanelHeight-2)
	case keyVimDown:
		m.detail.scroll = m.detail.scroll.down(len(m.detail.lines), detailPanelHeight-2)
	case keyVimUp:
		m.detail.scroll = m.detail.scroll.up()
	}
	return m, nil
}

func (m App) handleFilterKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.Code {
	case tea.KeyEsc, tea.KeyEnter:
		m.filtering = false
	case tea.KeyBackspace, tea.KeyDelete:
		if len(m.filterQuery) > 0 {
			selectedKey := m.currentSelectedKey()
			runes := []rune(m.filterQuery)
			m.filterQuery = string(runes[:len(runes)-1])
			m = m.rebuildTable(selectedKey)
		}
	default:
		if len(msg.Text) > 0 {
			selectedKey := m.currentSelectedKey()
			m.filterQuery += msg.Text
			m = m.rebuildTable(selectedKey)
		}
	}
	return m, nil
}

func (m App) handleConfirmKey(msg tea.KeyPressMsg, inProgressOp Operation, bulkCmdFn func(gh.PR) tea.Cmd, singleCmdFn func() tea.Cmd) (tea.Model, tea.Cmd) {
	switch msg.Code {
	case 'y', 'Y':
		m.op = inProgressOp
		if sel := m.selectedPRs(); len(sel) > 0 && bulkCmdFn != nil {
			cmds := make([]tea.Cmd, len(sel))
			for i, pr := range sel {
				cmds[i] = bulkCmdFn(pr)
			}
			m.bulkTotal = len(sel)
			m.bulkPending = len(sel)
			m.bulkFailed = 0
			if inProgressOp == OpApproving {
				m.bulkApproved = nil
			}
			return m, tea.Batch(cmds...)
		}
		return m, singleCmdFn()
	case 'n', 'N', tea.KeyEsc:
		m.op = OpNone
	}
	return m, nil
}

func (m App) handleConfirmApproveKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	return m.handleConfirmKey(msg, OpApproving,
		func(pr gh.PR) tea.Cmd { return m.client.ApprovePR(pr.Number, pr.Repository.NameWithOwner) },
		func() tea.Cmd { return m.client.ApprovePR(m.confirmNum, m.confirmRepo) },
	)
}

func (m App) handleConfirmCloseKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	return m.handleConfirmKey(msg, OpClosing,
		func(pr gh.PR) tea.Cmd { return m.client.ClosePR(pr.Number, pr.Repository.NameWithOwner) },
		func() tea.Cmd { return m.client.ClosePR(m.confirmNum, m.confirmRepo) },
	)
}

func (m App) handleConfirmMergeKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	return m.handleConfirmKey(msg, OpMerging,
		func(pr gh.PR) tea.Cmd { return m.client.MergePR(pr.Number, pr.Repository.NameWithOwner, "squash") },
		func() tea.Cmd { return m.client.MergePR(m.confirmNum, m.confirmRepo, "squash") },
	)
}

func (m App) handleConfirmAutoMergeKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	return m.handleConfirmKey(msg, OpAutoMerging,
		func(pr gh.PR) tea.Cmd { return m.client.AutoMergePR(pr.Number, pr.Repository.NameWithOwner, "squash") },
		func() tea.Cmd { return m.client.AutoMergePR(m.confirmNum, m.confirmRepo, "squash") },
	)
}

func (m App) handleConfirmUpdateKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	return m.handleConfirmKey(msg, OpUpdating, nil,
		func() tea.Cmd { return m.client.UpdateBranch(m.confirmNum, m.confirmRepo) },
	)
}

func (m App) handleConfirmRerunKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	return m.handleConfirmKey(msg, OpRerunning, nil,
		func() tea.Cmd { return m.client.RerunChecks(m.confirmNum, m.confirmRepo) },
	)
}

func bulkSummary(verb string, total, failed int) string {
	succeeded := total - failed
	if failed > 0 {
		return strconv.Itoa(succeeded) + " " + verb + ", " + strconv.Itoa(failed) + " failed"
	}
	return "done: " + verb + " " + strconv.Itoa(total) + " PRs"
}
