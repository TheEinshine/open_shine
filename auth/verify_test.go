package auth

import (
	"testing"
	"time"
)

func TestPasswordRoundtrip(t *testing.T) {
	h, err := Hash("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if !Verify(h, "correct horse battery staple") {
		t.Error("correct password did not verify")
	}
	if Verify(h, "wrong password") {
		t.Error("wrong password verified")
	}
}

func TestTokenUniqueAndLength(t *testing.T) {
	a, err := Token()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := Token()
	if a == b {
		t.Error("two tokens were identical")
	}
	if len(a) != 64 {
		t.Errorf("token length = %d, want 64", len(a))
	}
}

func TestLimiterLockout(t *testing.T) {
	l := newLimiter(3, time.Minute)
	ip := "1.2.3.4"
	if l.locked(ip) {
		t.Fatal("fresh ip should not be locked")
	}
	l.fail(ip)
	l.fail(ip)
	if l.locked(ip) {
		t.Fatal("locked too early")
	}
	l.fail(ip) // 3rd failure -> lock
	if !l.locked(ip) {
		t.Fatal("should be locked after max failures")
	}
	l.reset(ip)
	if l.locked(ip) {
		t.Fatal("reset should clear lock")
	}
}
