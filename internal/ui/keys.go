package ui

import tea "charm.land/bubbletea/v2"

const (
	keyQuit         = "q"
	keyForceQuit    = "ctrl+c"
	keyRefresh      = "r"
	keyToggleAll    = "A"
	keyFilter       = "/"
	keyDetail       = "v"
	keyApprove      = "a"
	keyMerge        = "m"
	keyClose        = "C"
	keyBrowser      = "o"
	keyCopy         = "c"
	keySelect       = "x"
	keyUpdate       = "u"
	keyVimDown      = "j"
	keyVimUp        = "k"
	keyScrollBottom = "G"
)

func isFilterKey(msg tea.KeyPressMsg) bool {
	switch msg.Code {
	case tea.KeyEsc, tea.KeyEnter, tea.KeyBackspace, tea.KeyDelete:
		return true
	}
	return len(msg.Text) > 0
}
