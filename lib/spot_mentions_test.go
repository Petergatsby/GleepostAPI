package lib

import (
	"testing"

	"github.com/Petergatsby/GleepostAPI/lib/conf"
	"github.com/Petergatsby/GleepostAPI/lib/gp"
)

func TestSpotMentions(t *testing.T) {
	config := conf.GetConfig()
	api := New(*config)
	type mentionTest struct {
		message          string
		conversationID   gp.ConversationID
		expectedMentions []gp.UserID
	}
	tests := []mentionTest{
		{
			message:        "hello",
			conversationID: 1,
		},
		{
			message:          "hi <@1|patrick>",
			conversationID:   1,
			expectedMentions: []gp.UserID{1},
		},
		{
			message:          "hi <@1|patrick> and <@2|dominic>",
			conversationID:   1,
			expectedMentions: []gp.UserID{1, 2},
		},
		{
			message:          "hi <@1|patrick> and <@2|dominic> and <@3|someone>",
			conversationID:   1,
			expectedMentions: []gp.UserID{1, 2},
		},
		{
			message:          "hi <@1|patrick> and <@1|patrick>",
			conversationID:   1,
			expectedMentions: []gp.UserID{1},
		},
		{
			message:          "hi <@all|@all>",
			conversationID:   1,
			expectedMentions: []gp.UserID{1, 2},
		},
	}
	for _, test := range tests {
		mentioned := api.spotMentions(test.message, test.conversationID)
		if len(mentioned) != len(test.expectedMentions) {
			t.Fatalf("Number of mentioned users didn't match expectations. (%d) vs expected (%d)\n", len(mentioned), len(test.expectedMentions))
		}
		for _, u := range test.expectedMentions {
			if !mentioned.Contains(u) {
				t.Fatal("Mentions didn't match")
			}
		}
	}
}
