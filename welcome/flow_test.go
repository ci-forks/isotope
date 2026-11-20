package main

import (
	"reflect"
	"testing"
)

func TestPagesCosmic(t *testing.T) {
	want := []PageID{PWelcome, PLanguage, PNetwork, PAppearance, PLayout, PWorkflow, PLauncher, PAccessibility, PApps, PShortcuts, PReady}
	got := Pages(Cosmic)
	if len(got) != 11 || !reflect.DeepEqual(got, want) {
		t.Fatalf("Pages(Cosmic) = %v, want %v", got, want)
	}
}

func TestPagesTiling(t *testing.T) {
	want := []PageID{PWelcome, PLanguage, PNetwork, PAppearance, PAccessibility, PApps, PShortcuts, PReady}
	for _, d := range []Desktop{Sway, I3, Niri} {
		got := Pages(d)
		if len(got) != 8 || !reflect.DeepEqual(got, want) {
			t.Errorf("Pages(%s) = %v, want %v", d, got, want)
		}
	}
}

func TestPageTitle(t *testing.T) {
	if PageTitle(PAccessibility) != "Accessibility" || PageTitle(PReady) != "Ready" {
		t.Fatal("unexpected titles")
	}
}

func TestFlowBackAtStart(t *testing.T) {
	f := NewFlow(Sway)
	f.Back()
	if f.Index() != 0 || f.Current() != PWelcome {
		t.Fatalf("index %d current %s", f.Index(), f.Current())
	}
}

func TestFlowNextBack(t *testing.T) {
	f := NewFlow(Sway)
	f.Next()
	if f.Current() != PLanguage {
		t.Fatalf("current %s", f.Current())
	}
	f.Back()
	if f.Current() != PWelcome {
		t.Fatalf("current %s", f.Current())
	}
}

func TestFlowSkipRecords(t *testing.T) {
	f := NewFlow(Cosmic)
	f.Next() // Language
	f.Skip()
	if f.Current() != PNetwork {
		t.Fatalf("current %s", f.Current())
	}
	if !reflect.DeepEqual(f.Outcome.Skipped, []PageID{PLanguage}) {
		t.Fatalf("skipped = %v", f.Outcome.Skipped)
	}
}

func TestFlowEndsAtReady(t *testing.T) {
	f := NewFlow(Niri)
	for i := 0; i < 20 && !f.Done(); i++ {
		f.Next()
	}
	if !f.Done() || f.Current() != PReady {
		t.Fatalf("done %v current %s", f.Done(), f.Current())
	}
}

func TestSingle(t *testing.T) {
	f := Single(PApps)
	if f.Current() != PApps || f.Index() != 0 || f.Done() {
		t.Fatalf("bad start: %s %d", f.Current(), f.Index())
	}
	f.Next()
	if !f.Done() {
		t.Fatal("Next should end a single-page flow")
	}
	if f.Current() != PApps {
		t.Fatalf("current %s", f.Current())
	}
}
