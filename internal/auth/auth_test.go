package auth

import (
	"strings"
	"testing"
	"time"
)

func TestHashAndVerify(t *testing.T) {
	encoded, err := Hash("doğru-at-pili-zımba")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(encoded, "$argon2id$") || strings.Contains(encoded, "doğru") {
		t.Fatalf("encoded = %s", encoded)
	}
	if !Verify("doğru-at-pili-zımba", encoded) {
		t.Fatal("right password rejected")
	}
	if Verify("yanlış-at-pili-zımba", encoded) {
		t.Fatal("wrong password accepted")
	}
	other, _ := Hash("doğru-at-pili-zımba")
	if other == encoded {
		t.Fatal("salt is not random")
	}
}

func TestVerifyRejectsGarbage(t *testing.T) {
	for _, encoded := range []string{"", "plain", "$argon2id$v=19$bad$x$y", "$bcrypt$a$b$c$d"} {
		if Verify("x", encoded) {
			t.Fatalf("accepted %q", encoded)
		}
	}
}

func TestValidPasswordCountsCharacters(t *testing.T) {
	if ValidPassword("kısa") || !ValidPassword("şşşşşşşşşşşş") || ValidPassword("şşşşşşşşşşş") {
		t.Fatal("length rule is wrong")
	}
}

func TestTokenHashIsStable(t *testing.T) {
	token, hash, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	if HashToken(token) != hash || token == hash || len(token) != 64 {
		t.Fatalf("%s %s", token, hash)
	}
}

func TestLimiterLocksAndDoubles(t *testing.T) {
	clock := time.Unix(1000, 0)
	limiter := NewLimiter()
	limiter.now = func() time.Time { return clock }
	for range freeAttempts - 1 {
		limiter.Fail("a")
	}
	if ok, _ := limiter.Allow("a"); !ok {
		t.Fatal("locked too early")
	}
	limiter.Fail("a")
	ok, wait := limiter.Allow("a")
	if ok || wait != time.Minute {
		t.Fatalf("%v %v", ok, wait)
	}
	if ok, _ := limiter.Allow("b"); !ok {
		t.Fatal("another client was locked")
	}
	clock = clock.Add(time.Minute)
	if ok, _ := limiter.Allow("a"); !ok {
		t.Fatal("lock did not expire")
	}
	limiter.Fail("a")
	if _, wait := limiter.Allow("a"); wait != 2*time.Minute {
		t.Fatalf("wait = %v", wait)
	}
	limiter.Reset("a")
	if ok, _ := limiter.Allow("a"); !ok {
		t.Fatal("reset did not clear the lock")
	}
}

func TestLimiterCapsTheLock(t *testing.T) {
	clock := time.Unix(1000, 0)
	limiter := NewLimiter()
	limiter.now = func() time.Time { return clock }
	for range 30 {
		limiter.Fail("a")
	}
	if _, wait := limiter.Allow("a"); wait != lockMax {
		t.Fatalf("wait = %v", wait)
	}
}
