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
