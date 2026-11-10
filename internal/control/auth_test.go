package control

import (
	"errors"
	"testing"
	"time"
)

func TestAuthorizeOneGUIUser(t *testing.T) {
	got, ok := Authorize(Auth{}, true, 1000, "/usr/bin/MultiRoute", "/usr/bin/MultiRoute")
	if !ok || got.UID != 1000 || got.Exe != "/usr/bin/MultiRoute" {
		t.Fatalf("%+v %v", got, ok)
	}
	if _, ok := Authorize(Auth{}, true, 0, "/usr/bin/MultiRoute", "/usr/bin/MultiRoute"); ok {
		t.Fatal("root must not claim the desktop")
	}
	if _, ok := Authorize(Auth{}, true, 1000, "/tmp/other", "/usr/bin/MultiRoute"); ok {
		t.Fatal("unexpected executable claimed the service")
	}
	if _, ok := Authorize(got, false, 1001, got.Exe, got.Exe); ok {
		t.Fatal("second user was accepted")
	}
	if _, ok := Authorize(got, false, got.UID, "/usr/bin/other", got.Exe); ok {
		t.Fatal("replacement executable was accepted")
	}
	if _, ok := Authorize(got, false, got.UID, got.Exe+" (deleted)", got.Exe); ok {
		t.Fatal("deleted executable was accepted")
	}
}

func TestAuthFileRoundTrip(t *testing.T) {
	text := FormatAuth(Auth{UID: 1000, Exe: "/usr/bin/MultiRoute"})
	got, err := ParseAuth(text)
	if err != nil || got.UID != 1000 || got.Exe != "/usr/bin/MultiRoute" {
		t.Fatal(err, got)
	}
	if _, err := ParseAuth("uid=1\nexe=/bin/sh\nrun=reboot\n"); err == nil {
		t.Fatal("extra command accepted")
	}
}

func TestServiceKind(t *testing.T) {
	if got := ServiceKind(true, ErrAuthorization); got != ServiceAuthorization {
		t.Fatal(got)
	}
	if got := ServiceKind(true, errors.New("broken")); got != ServiceUnreachable {
		t.Fatal(got)
	}
	if got := ServiceKind(true, nil); got != "" {
		t.Fatal(got)
	}
}

func TestSampleInterval(t *testing.T) {
	if SampleInterval != 70*time.Millisecond {
		t.Fatal(SampleInterval)
	}
}
