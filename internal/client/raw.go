package client

import (
	"encoding/json"

	"github.com/mudrii/gobird/internal/types"
)

func attachRawToTweets(items []types.TweetData, body []byte) []types.TweetData {
	if len(items) == 0 || len(body) == 0 {
		return items
	}
	var raw any
	if err := json.Unmarshal(body, &raw); err != nil {
		return items
	}
	for i := range items {
		items[i].Raw = raw
	}
	return items
}

func attachRawToNews(items []types.NewsItem, body []byte) []types.NewsItem {
	if len(items) == 0 || len(body) == 0 {
		return items
	}
	var raw any
	if err := json.Unmarshal(body, &raw); err != nil {
		return items
	}
	for i := range items {
		items[i].Raw = raw
	}
	return items
}

func attachRawToUsers(items []types.TwitterUser, body []byte) []types.TwitterUser {
	if len(items) == 0 || len(body) == 0 {
		return items
	}
	var raw any
	if err := json.Unmarshal(body, &raw); err != nil {
		return items
	}
	for i := range items {
		items[i].Raw = raw
	}
	return items
}

func attachRawToLists(items []types.TwitterList, body []byte) []types.TwitterList {
	if len(items) == 0 || len(body) == 0 {
		return items
	}
	var raw any
	if err := json.Unmarshal(body, &raw); err != nil {
		return items
	}
	for i := range items {
		items[i].Raw = raw
	}
	return items
}
