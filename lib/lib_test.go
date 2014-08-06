package lib

import "testing"

func TestNormalizeEmail(t *testing.T) {
	target := "patrick@gleepost.com"
	cheeky := "patrick+evadingaban@gleepost.com"
	normalized := normalizeEmail(cheeky)
	if normalized != target {
		t.Fail()
	}
}
