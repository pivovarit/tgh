package ui

type scrollState struct {
	offset int
}

func (s scrollState) up() scrollState {
	if s.offset > 0 {
		s.offset--
	}
	return s
}

func (s scrollState) down(contentLen, viewHeight int) scrollState {
	maxOff := max(0, contentLen-viewHeight)
	if s.offset < maxOff {
		s.offset++
	}
	return s
}

func (s scrollState) top() scrollState {
	s.offset = 0
	return s
}

func (s scrollState) bottom(contentLen, viewHeight int) scrollState {
	s.offset = max(0, contentLen-viewHeight)
	return s
}
