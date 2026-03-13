// Package bird provides a public Go client for the Twitter/X API.
//
// Create a client with New or NewWithTokens. All networked operations accept a
// context.Context as the first argument.
//
// Some read methods return a result struct with embedded status fields instead
// of a separate error return. For those methods, check result.Error and
// result.Success.
//
// Example:
//
//	creds, err := bird.ResolveCredentials(bird.ResolveOptions{})
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	client, err := bird.New(creds, nil)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	page := client.Search(ctx, "golang", &bird.SearchOptions{Product: "Latest"})
//	if page.Error != nil {
//		log.Fatal(page.Error)
//	}
//
//	for _, tweet := range page.Items {
//		log.Println(tweet.ID, tweet.Text)
//	}
//
//	tweet, err := client.GetTweet(ctx, "12345", nil)
//	if err != nil {
//		log.Fatal(err)
//	}
//	log.Println(tweet.Author.Username, tweet.Text)
package bird
