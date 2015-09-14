package lib

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
	"github.com/mattbaird/elastigo/lib"
)

//MessageResult contains one or more messages which matched a search query, plus their context (a few messages before and after the hit)
type MessageResult struct {
	Messages []MatchedMessage `json:"messages"`
}

//MatchedMessage is a message which (maybe) was a match in a query
type MatchedMessage struct {
	gp.Message
	Matched bool `json:"matched,omitempty"`
}

//SearchMessagesInConversation does exactly what it says on the tin.
func (api *API) SearchMessagesInConversation(userID gp.UserID, convID gp.ConversationID, query string, mode int, index int64) (hits []MessageResult, err error) {
	hits = make([]MessageResult, 0)
	if !api.userCanViewConversation(userID, convID) {
		return hits, ENOTALLOWED
	}
	threshold, err := api.getDeletionThreshold(userID, convID)
	if err != nil {
		return hits, err
	}
	messages, err := api.esSearchConversation(convID, query, threshold)
	if err != nil {
		log.Println(err)
		return
	}
	for _, message := range messages {
		var before, since []gp.Message
		before, err = api.getMessages(userID, convID, ChronologicallyBeforeID, int64(message.Message.ID), 2)
		if err != nil {
			return
		}
		since, err = api.getMessages(userID, convID, ChronologicallyAfterID, int64(message.Message.ID), 2)
		if err != nil {
			return
		}
		context := []MatchedMessage{}
		for _, msg := range since {
			context = append(context, MatchedMessage{Message: msg})
		}
		context = append(context, MatchedMessage{Message: message.Message, Matched: true})
		for _, msg := range before {
			context = append(context, MatchedMessage{Message: msg})
		}
		result := MessageResult{Messages: context}
		hits = append(hits, result)
	}
	return hits, nil
}

type esMessage struct {
	gp.Message
	ConvID gp.ConversationID `json:"conversation"`
}

func (api *API) esIndexMessage(message gp.Message, conversation gp.ConversationID) {
	msg := esMessage{Message: message, ConvID: conversation}
	c := elastigo.NewConn()
	c.Domain = api.Config.ElasticSearch
	msg.Time = msg.Time.Round(time.Second)
	c.Index("gleepost", "messages", fmt.Sprintf("%d", msg.ID), nil, msg)
	return
}

func (api *API) esSearchConversation(convID gp.ConversationID, query string, threshold gp.MessageID) (messages []esMessage, err error) {
	c := elastigo.NewConn()
	c.Domain = api.Config.ElasticSearch
	esQuery := esgroupquery{}
	//Restrict to this conversation
	conversationTerm := make(map[string]string)
	conversationTerm["conversation"] = fmt.Sprintf("%d", convID)
	//Restrict to non-deleted messages
	deletionThreshold := make(map[string]interface{})
	deletionThreshold["id"] = struct {
		Threshold gp.MessageID `json:"gt"`
	}{Threshold: threshold}

	esQuery.Query.Filtered.Filter.Bool.Must = []interface{}{term{T: conversationTerm}, rangeFilter{R: deletionThreshold}}
	fields := []string{"text"}
	for _, field := range fields {
		match := make(map[string]string)
		matcher := matcher{Match: match}
		matcher.Match[field] = query
		esQuery.Query.Filtered.Query.Bool.Should = append(esQuery.Query.Filtered.Query.Bool.Should, matcher)
	}
	q, _ := json.Marshal(esQuery)
	log.Printf("%s", q)
	results, err := c.Search("gleepost", "messages", nil, esQuery)
	if err != nil {
		return
	}
	for _, hit := range results.Hits.Hits {
		var message esMessage
		err = json.Unmarshal(*hit.Source, &message)
		if err != nil {
			return
		}
		messages = append(messages, message)
	}
	return
}
