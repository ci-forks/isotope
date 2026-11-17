package main

import (
	"testing"

	"github.com/kairos-community/isotope/install-ui/core"
)

// These need no GPU: they drive the UI's page logic over core.

func TestBackWalksThePagesInOrder(t *testing.T) {
	u := fixture(t, false)
	toSummary(t, u)
	for _, want := range []gpage{pEncrypt, pDisk, pAccount, pKeyboard, pWelcome, pWelcome} {
		u.prev()
		if u.page != want {
			t.Fatalf("back went to %d, want %d", u.page, want)
		}
	}
}

func TestSummaryChangeAndForwardAgain(t *testing.T) {
	u := fixture(t, false)
	toSummary(t, u)
	u.w.Page = u.firstCorePage(pAccount)
	u.goTo(pAccount)
	set(u.user, "sam")
	mustNext(t, u, pDisk)
	mustNext(t, u, pEncrypt)
	mustNext(t, u, pRecovery)
	set(u.rk, "vutrnlkj")
	mustNext(t, u, pSummary)
	if u.w.Choice.Username != "sam" || u.w.Choice.RecoveryKeyValue != testKey {
		t.Fatalf("choice after the change: %+v", u.w.Choice)
	}
}

func TestPasswordsMustMatch(t *testing.T) {
	u := fixture(t, false)
	toAccount(t, u)
	set(u.pw2, "something else")
	if u.passwordsMatch() {
		t.Fatal("different passwords match")
	}
	u.next()
	if u.page != pAccount || u.pw2.err == "" {
		t.Fatalf("page %d, error %q", u.page, u.pw2.err)
	}
}

func TestCoreErrorsShowUnderTheirFields(t *testing.T) {
	u := fixture(t, false)
	toAccount(t, u)
	set(u.user, "Alex")
	set(u.pw, "")
	set(u.pw2, "")
	u.next()
	if u.page != pAccount {
		t.Fatal("left the page with errors")
	}
	if u.user.err == "" || u.pw.err != "Password is required." {
		t.Fatalf("errors: user %q, password %q", u.user.err, u.pw.err)
	}
}

func TestEncryptOffSkipsTheKey(t *testing.T) {
	u := fixture(t, false)
	toEncrypt(t, u)
	u.encOn = false
	mustNext(t, u, pSummary)
	if u.w.Choice.Encrypt || u.w.Choice.RecoveryKey {
		t.Fatalf("choice: %+v", u.w.Choice)
	}
}

func TestShortPassphraseStays(t *testing.T) {
	u := fixture(t, false)
	toEncrypt(t, u)
	set(u.pp, "short")
	set(u.pp2, "short")
	u.next()
	if u.page != pEncrypt || u.pp.err == "" {
		t.Fatalf("page %d, error %q", u.page, u.pp.err)
	}
}

func TestRecoveryKeyContinueNeedsTheGroup(t *testing.T) {
	u := fixture(t, false)
	toRecovery(t, u)
	if u.recoveryConfirmed() {
		t.Fatal("a part of the group confirms")
	}
	set(u.rk, "vutrnlkj")
	if !u.recoveryConfirmed() {
		t.Fatal("the group does not confirm")
	}
	if u.w.Page != core.PageRecoveryConfirm {
		t.Fatal("checking the group moved the wizard")
	}
}

func TestTrustedBootAsksOnlyForTheKey(t *testing.T) {
	u := fixture(t, true)
	toAccount(t, u)
	mustNext(t, u, pDisk)
	mustNext(t, u, pEncrypt)
	mustNext(t, u, pSummary)
	if u.w.Choice.Encrypt || u.w.Choice.RecoveryKey {
		t.Fatalf("choice: %+v", u.w.Choice)
	}
}

func TestNewLayoutRelaunches(t *testing.T) {
	u := fixture(t, false)
	var got []core.GUIState
	u.hooks.relaunch = func(s core.GUIState) { got = append(got, s) }
	u.canRelaunch = true
	toKeyboard(t, u)
	u.pickKeyboard(u.kbSel)
	set(u.host, "box")
	u.next()
	if len(got) != 2 {
		t.Fatalf("%d relaunches, want 2", len(got))
	}
	if got[0].Layout != "it" || got[0].Next != "keyboard" || got[1].Next != "account" {
		t.Fatalf("states: %+v", got)
	}

	// Same layout as cage: no relaunch.
	u2 := fixture(t, false)
	u2.canRelaunch = true
	u2.runLayout = "it"
	toKeyboard(t, u2)
	mustNext(t, u2, pAccount)
}

func TestRestoreReopensWhereItWas(t *testing.T) {
	u := fixture(t, false)
	u.restore(core.GUIState{Next: "account", Layout: "it", Timezone: "Europe/Paris", Hostname: "box", Username: "alex", Github: "alex", Disk: "/dev/sda", Encrypt: true})
	if u.page != pAccount || u.w.Choice.KeyboardLayout != "it" || u.w.Choice.Timezone != "Europe/Paris" {
		t.Fatalf("page %d, choice %+v", u.page, u.w.Choice)
	}
	if u.user.ed.Text() != "alex" || u.pw.ed.Text() != "" || u.diskSel != 1 || !u.encOn {
		t.Fatal("fields not restored")
	}
}

func TestSizeText(t *testing.T) {
	for gb, want := range map[int]string{64: "64 GB", 1000: "1 TB", 2000: "2 TB", 1500: "1.5 TB", 12000: "12 TB"} {
		if got := sizeText(gb); got != want {
			t.Errorf("sizeText(%d) = %q, want %q", gb, got, want)
		}
	}
}
