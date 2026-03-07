package ui

import "testing"

func TestScrollState_Up(t *testing.T) {
	s := scrollState{offset: 5}
	s = s.up()
	if s.offset != 4 {
		t.Errorf("up() offset = %d, want 4", s.offset)
	}

	s = scrollState{offset: 0}
	s = s.up()
	if s.offset != 0 {
		t.Errorf("up() from 0 should stay 0, got %d", s.offset)
	}
}

func TestScrollState_Down(t *testing.T) {
	s := scrollState{offset: 0}
	s = s.down(20, 10)
	if s.offset != 1 {
		t.Errorf("down() offset = %d, want 1", s.offset)
	}

	s = scrollState{offset: 10}
	s = s.down(20, 10)
	if s.offset != 10 {
		t.Errorf("down() at max should stay 10, got %d", s.offset)
	}
}

func TestScrollState_Top(t *testing.T) {
	s := scrollState{offset: 5}
	s = s.top()
	if s.offset != 0 {
		t.Errorf("top() offset = %d, want 0", s.offset)
	}
}

func TestScrollState_Bottom(t *testing.T) {
	s := scrollState{offset: 0}
	s = s.bottom(20, 10)
	if s.offset != 10 {
		t.Errorf("bottom() offset = %d, want 10", s.offset)
	}

	s = scrollState{offset: 0}
	s = s.bottom(5, 10)
	if s.offset != 0 {
		t.Errorf("bottom() with short content offset = %d, want 0", s.offset)
	}
}
