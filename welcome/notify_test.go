package main

import (
	"strings"
	"testing"
	"time"

	"github.com/godbus/dbus/v5"
)

func TestCompletionNotice(t *testing.T) {
	s, b := CompletionNotice([]string{"Firefox", "VLC"}, nil)
	if s != "Apps installed" || !strings.Contains(b, "Installed: Firefox, VLC.") {
		t.Errorf("%q %q", s, b)
	}
	s, b = CompletionNotice([]string{"Firefox"}, []string{"VLC"})
	if s != "Some apps did not install" || !strings.Contains(b, "Failed: VLC.") {
		t.Errorf("%q %q", s, b)
	}
	s, b = CompletionNotice(nil, []string{"VLC"})
	if s != "Apps did not install" || !strings.Contains(b, "Failed: VLC.") {
		t.Errorf("%q %q", s, b)
	}
}

func TestNotifySendsBusctlCall(t *testing.T) {
	r := &fakeRunner{}
	if err := Notify(r, "Apps installed", "Done.", 1, 0); err != nil {
		t.Fatal(err)
	}
	c := r.called()
	if len(c) != 1 || !strings.HasPrefix(c[0], "busctl --user call org.freedesktop.Notifications") || !strings.Contains(c[0], "Apps installed Done.") {
		t.Errorf("%v", c)
	}
	r = &fakeRunner{fail: []string{"busctl"}}
	if err := Notify(r, "a", "b", 3, 0); err == nil || len(r.called()) != 3 {
		t.Errorf("want 3 tries and an error, got %v %v", len(r.called()), err)
	}
}

func sig(name string, body ...any) *dbus.Signal {
	return &dbus.Signal{Name: "org.freedesktop.Notifications." + name, Body: body}
}

func TestWaitAction(t *testing.T) {
	c := make(chan *dbus.Signal, 4)
	c <- sig("ActionInvoked", uint32(8), "default") // another notification
	c <- sig("ActionInvoked", uint32(7), "default")
	if !waitAction(c, 7, time.Second) {
		t.Error("the default action of our notification must count")
	}
	c <- sig("NotificationClosed", uint32(7), uint32(1))
	if waitAction(c, 7, time.Second) {
		t.Error("a close without a click is not a click")
	}
	if waitAction(make(chan *dbus.Signal), 7, 10*time.Millisecond) {
		t.Error("a timeout is not a click")
	}
}
