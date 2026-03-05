package ui

import (
	tea "charm.land/bubbletea/v2"
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
		cursor := m.table.Cursor()
		filtered := m.filteredPRs()
		if cursor >= 0 && cursor < len(filtered) {
			pr := filtered[cursor]
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
		} else {
			cursor := m.table.Cursor()
			filtered := m.filteredPRs()
			if cursor >= 0 && cursor < len(filtered) {
				pr := filtered[cursor]
				m.op = OpConfirmApprove
				m.confirmNum = pr.Number
				m.confirmTitle = pr.Title
				m.confirmRepo = pr.Repository.NameWithOwner
			}
		}
		return m, nil
	case keyClose:
		if sel := m.selectedPRs(); len(sel) > 0 {
			m.op = OpConfirmClose
			m.confirmNum = 0
			m.confirmTitle = ""
			m.confirmRepo = ""
		} else {
			cursor := m.table.Cursor()
			filtered := m.filteredPRs()
			if cursor >= 0 && cursor < len(filtered) {
				pr := filtered[cursor]
				m.op = OpConfirmClose
				m.confirmNum = pr.Number
				m.confirmTitle = pr.Title
				m.confirmRepo = pr.Repository.NameWithOwner
			}
		}
		return m, nil
	case keyMerge:
		if sel := m.selectedPRs(); len(sel) > 0 {
			m.op = OpConfirmMerge
			m.confirmNum = 0
			m.confirmTitle = ""
			m.confirmRepo = ""
		} else {
			cursor := m.table.Cursor()
			filtered := m.filteredPRs()
			if cursor >= 0 && cursor < len(filtered) {
				pr := filtered[cursor]
				m.op = OpConfirmMerge
				m.confirmNum = pr.Number
				m.confirmTitle = pr.Title
				m.confirmRepo = pr.Repository.NameWithOwner
			}
		}
		return m, nil
	case keyBrowser:
		cursor := m.table.Cursor()
		filtered := m.filteredPRs()
		if cursor >= 0 && cursor < len(filtered) {
			pr := filtered[cursor]
			return m, m.client.OpenBrowser(pr.Number, pr.Repository.NameWithOwner)
		}
	case keyCopy:
		cursor := m.table.Cursor()
		filtered := m.filteredPRs()
		if cursor >= 0 && cursor < len(filtered) {
			pr := filtered[cursor]
			return m, copyToClipboard(pr.Title, pr.URL)
		}
	default:
		if msg.Code == tea.KeyEsc && m.op == OpNone {
			if len(m.selected) > 0 {
				m.selected = nil
				m = m.refreshTableRows()
				return m, nil
			} else if m.filterQuery != "" {
				selectedNum := m.currentSelectedNumber()
				m.filterQuery = ""
				m = m.rebuildTable(selectedNum)
			} else if m.err != nil {
				m.err = nil
			}
		}
	}

	var tableMsg tea.Msg = msg
	switch msg.Text {
	case keyVimDown:
		tableMsg = tea.KeyPressMsg{Code: tea.KeyDown}
	case keyVimUp:
		tableMsg = tea.KeyPressMsg{Code: tea.KeyUp}
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(tableMsg)
	cursor := m.table.Cursor()
	height := m.tableHeight()
	if cursor < m.viewportStart {
		m.viewportStart = cursor
	} else if height > 0 && cursor >= m.viewportStart+height {
		m.viewportStart = cursor - height + 1
	}
	return m, cmd
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
			selectedNum := m.currentSelectedNumber()
			runes := []rune(m.filterQuery)
			m.filterQuery = string(runes[:len(runes)-1])
			m = m.rebuildTable(selectedNum)
		}
	default:
		if len(msg.Text) > 0 {
			selectedNum := m.currentSelectedNumber()
			m.filterQuery += msg.Text
			m = m.rebuildTable(selectedNum)
		}
	}
	return m, nil
}

func (m App) handleConfirmCloseKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.Code {
	case 'y', 'Y':
		m.op = OpClosing
		if sel := m.selectedPRs(); len(sel) > 0 {
			cmds := make([]tea.Cmd, len(sel))
			for i, pr := range sel {
				cmds[i] = m.client.ClosePR(pr.Number, pr.Repository.NameWithOwner)
			}
			m.bulkTotal = len(sel)
			m.bulkPending = len(sel)
			m.bulkFailed = 0
			return m, tea.Batch(cmds...)
		}
		return m, m.client.ClosePR(m.confirmNum, m.confirmRepo)
	case 'n', 'N', tea.KeyEsc:
		m.op = OpNone
	}
	return m, nil
}

func (m App) handleConfirmApproveKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.Code {
	case 'y', 'Y':
		m.op = OpApproving
		if sel := m.selectedPRs(); len(sel) > 0 {
			cmds := make([]tea.Cmd, len(sel))
			for i, pr := range sel {
				cmds[i] = m.client.ApprovePR(pr.Number, pr.Repository.NameWithOwner)
			}
			m.bulkTotal = len(sel)
			m.bulkPending = len(sel)
			m.bulkFailed = 0
			return m, tea.Batch(cmds...)
		}
		return m, m.client.ApprovePR(m.confirmNum, m.confirmRepo)
	case 'n', 'N', tea.KeyEsc:
		m.op = OpNone
	}
	return m, nil
}

func (m App) handleConfirmMergeKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.Code {
	case 'y', 'Y':
		m.op = OpMerging
		if sel := m.selectedPRs(); len(sel) > 0 {
			cmds := make([]tea.Cmd, len(sel))
			for i, pr := range sel {
				cmds[i] = m.client.MergePR(pr.Number, pr.Repository.NameWithOwner, "squash")
			}
			m.bulkTotal = len(sel)
			m.bulkPending = len(sel)
			m.bulkFailed = 0
			return m, tea.Batch(cmds...)
		}
		return m, m.client.MergePR(m.confirmNum, m.confirmRepo, "squash")
	case 'n', 'N', tea.KeyEsc:
		m.op = OpNone
	}
	return m, nil
}

func bulkSummary(verb string, total, failed int) string {
	succeeded := total - failed
	if failed > 0 {
		return itoa(succeeded) + " " + verb + ", " + itoa(failed) + " failed"
	}
	return "✓ " + verb + " " + itoa(total) + " PRs"
}

