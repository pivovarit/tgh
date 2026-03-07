package ui

func (m App) tableHeight() int {
	reserved := tableChrome
	if m.detail.visible {
		reserved += detailPanelHeight + 2
	}
	h := m.height - reserved
	if h < 3 {
		return 3
	}
	return h
}
