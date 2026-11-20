package guikit

import (
	"testing"

	"gioui.org/widget"
)

func TestFontsHaveTheThreeFaces(t *testing.T) {
	seen := map[string]bool{}
	for _, f := range Fonts() {
		seen[string(f.Font.Typeface)] = true
	}
	for _, face := range []string{string(FaceTitle), string(FaceBody), string(FaceMono)} {
		if !seen[face] {
			t.Errorf("no %s in the embedded fonts (have %v)", face, seen)
		}
	}
}

func TestOverMixesInSRGB(t *testing.T) {
	if got := Over(RGB(0x000000), RGB(0xffffff), 0.5); got != RGB(0x808080) {
		t.Fatalf("half white on black: %v", got)
	}
	if got := Over(ColWindow, ColAccent, 0); got != ColWindow {
		t.Fatalf("nothing laid on the window: %v", got)
	}
}

func TestMoveSelectionStepsThroughTheShownEntries(t *testing.T) {
	var l widget.List
	shown := []int{3, 5, 9}
	sel := MoveSelection(shown, -1, true, &l)
	if sel != 3 {
		t.Fatalf("from nothing: %d, want the first shown", sel)
	}
	for _, want := range []int{5, 9, 9} {
		if sel = MoveSelection(shown, sel, true, &l); sel != want {
			t.Fatalf("down: %d, want %d", sel, want)
		}
	}
	for _, want := range []int{5, 3, 3} {
		if sel = MoveSelection(shown, sel, false, &l); sel != want {
			t.Fatalf("up: %d, want %d", sel, want)
		}
	}
	if sel = MoveSelection(nil, 7, true, &l); sel != 7 {
		t.Fatalf("empty list moved the selection to %d", sel)
	}
}
