package main

import (
	"testing"
	"time"
)

func TestCreatePoll(t *testing.T) {
	err := initDB()
	if err != nil {
		t.Fatalf("Error initializing db: %v\n", err)
	}

	type createPollTest struct {
		Email              string
		Pass               string
		PollOptions        []string
		PollExpiry         string
		ExpectedStatusCode int
		ExpectedType       string
		ExpectedError      string
	}
	testGood := createPollTest{
		Email:              "patrick@fakestanford.edu",
		Pass:               "TestingPass",
		PollOptions:        []string{"Option 1", "Another option", "Nothing"},
		PollExpiry:         time.Now().Add(24 * time.Hour).String(),
		ExpectedStatusCode: 201,
		ExpectedType:       "Created",
	}

	tests := []createPollTest{testGood}
	for _, cpt := range tests {

	}
}
