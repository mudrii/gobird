package bird_test

import (
	"context"

	"github.com/mudrii/gobird/pkg/bird"
)

func ExampleNew() {
	creds := &bird.TwitterCookies{
		AuthToken: "auth_token",
		Ct0:       "ct0",
	}

	client, err := bird.New(creds, nil)
	if err != nil {
		panic(err)
	}

	_ = client
}

func ExampleClient_Search() {
	client, err := bird.NewWithTokens("auth_token", "ct0", nil)
	if err != nil {
		panic(err)
	}

	page := client.Search(context.Background(), "golang", &bird.SearchOptions{
		Product: "Latest",
	})
	if page.Error != nil {
		panic(page.Error)
	}

	_ = page.Items
}

func ExampleClient_GetTweet() {
	client, err := bird.NewWithTokens("auth_token", "ct0", nil)
	if err != nil {
		panic(err)
	}

	tweet, err := client.GetTweet(context.Background(), "12345", nil)
	if err != nil {
		panic(err)
	}

	_ = tweet
}

func ExampleNewWithTokens() {
	client, err := bird.NewWithTokens("my_auth_token", "my_ct0", nil)
	if err != nil {
		panic(err)
	}

	_ = client
}

func ExampleClient_GetHomeTimeline() {
	client, err := bird.NewWithTokens("auth_token", "ct0", nil)
	if err != nil {
		panic(err)
	}

	result := client.GetHomeTimeline(context.Background(), &bird.FetchOptions{
		Count: 20,
		Limit: 100,
	})
	if result.Error != nil {
		panic(result.Error)
	}

	_ = result.Items
}

func ExampleClient_GetUserTweets() {
	client, err := bird.NewWithTokens("auth_token", "ct0", nil)
	if err != nil {
		panic(err)
	}

	result, err := client.GetUserTweets(context.Background(), "12345", &bird.UserTweetsOptions{
		IncludeReplies: false,
	})
	if err != nil {
		panic(err)
	}

	_ = result.Items
}

func ExampleClient_Like() {
	client, err := bird.NewWithTokens("auth_token", "ct0", nil)
	if err != nil {
		panic(err)
	}

	if err := client.Like(context.Background(), "12345"); err != nil {
		panic(err)
	}
}

func ExampleClient_Tweet() {
	client, err := bird.NewWithTokens("auth_token", "ct0", nil)
	if err != nil {
		panic(err)
	}

	tweetID, err := client.Tweet(context.Background(), "Hello, world!")
	if err != nil {
		panic(err)
	}

	_ = tweetID
}
