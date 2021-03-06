package lib

import (
	"testing"
	"time"
)

const (
	testDb = "testdb"
)

func TestCreateToken(t *testing.T) {
	token := createToken(9)
	if token.UserID != 9 {
		t.Fail()
	}
	if len(token.Token) < 64 {
		t.Fail()
	}
	if !time.Now().Before(token.Expiry) {
		t.Fail()
	}
}

func BenchmarkCreateToken(b *testing.B) {
	for i := 0; i < b.N; i++ {
		token := createToken(9)
		if token.UserID != 9 {
			b.Fail()
		}
		if len(token.Token) < 64 {
			b.Fail()
		}
		if !time.Now().Before(token.Expiry) {
			b.Fail()
		}
	}
}

func TestLooksLikeEmail(t *testing.T) {
	couldBeEmail := looksLikeEmail("patrick@gleepost.com")
	if couldBeEmail != true {
		t.Fail()
	}
	couldBeEmail = looksLikeEmail("lol dongs")
	if couldBeEmail == true {
		t.Fail()
	}
	couldBeEmail = looksLikeEmail("@")
	if couldBeEmail == true {
		t.Fail()
	}
}

func BenchmarkLooksLikeEmail(b *testing.B) {
	for i := 0; i < b.N; i++ {
		couldBeEmail := looksLikeEmail("patrick@gleepost.com")
		if couldBeEmail != true {
			b.Fail()
		}
	}
}
