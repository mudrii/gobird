package parsing

import "github.com/mudrii/gobird/internal/types"

// AddThreadMetadata annotates tweets in a thread with position metadata.
func AddThreadMetadata(tweets []types.TweetData, _ string) []types.TweetWithMeta {
	if len(tweets) == 0 {
		return nil
	}

	rootID := tweets[0].ConversationID
	if rootID == "" {
		rootID = tweets[0].ID
	}

	out := make([]types.TweetWithMeta, 0, len(tweets))
	for _, t := range tweets {
		hasSelfReplies := false
		for _, candidate := range tweets {
			if candidate.AuthorID != t.AuthorID || candidate.InReplyToStatusID == nil {
				continue
			}
			if *candidate.InReplyToStatusID == t.ID {
				hasSelfReplies = true
				break
			}
		}

		isRoot := t.InReplyToStatusID == nil || *t.InReplyToStatusID == ""
		position := "middle"
		switch {
		case isRoot && !hasSelfReplies:
			position = "standalone"
		case isRoot:
			position = "root"
		case !hasSelfReplies:
			position = "end"
		}

		wm := types.TweetWithMeta{
			TweetData:      t,
			IsThread:       hasSelfReplies || !isRoot,
			ThreadPosition: position,
			HasSelfReplies: hasSelfReplies,
		}
		if rootID != "" {
			wm.ThreadRootID = &rootID
		}
		out = append(out, wm)
	}
	return out
}

// FilterAuthorChain keeps tweets that form a connected self-reply chain.
func FilterAuthorChain(tweets []types.TweetData, authorID string) []types.TweetData {
	if len(tweets) == 0 {
		return nil
	}
	byID := make(map[string]types.TweetData, len(tweets))
	children := map[string][]types.TweetData{}
	var anchor *types.TweetData
	for _, t := range tweets {
		byID[t.ID] = t
		if t.AuthorID == authorID && anchor == nil {
			tmp := t
			anchor = &tmp
		}
		if t.InReplyToStatusID != nil && *t.InReplyToStatusID != "" {
			children[*t.InReplyToStatusID] = append(children[*t.InReplyToStatusID], t)
		}
	}
	if anchor == nil {
		return nil
	}

	chainIDs := map[string]bool{anchor.ID: true}
	cur := *anchor
	for cur.InReplyToStatusID != nil && *cur.InReplyToStatusID != "" {
		parent, ok := byID[*cur.InReplyToStatusID]
		if !ok || parent.AuthorID != authorID {
			break
		}
		chainIDs[parent.ID] = true
		cur = parent
	}

	queue := []string{anchor.ID}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		for _, child := range children[id] {
			if child.AuthorID != authorID || chainIDs[child.ID] {
				continue
			}
			chainIDs[child.ID] = true
			queue = append(queue, child.ID)
		}
	}

	var out []types.TweetData
	for _, t := range tweets {
		if chainIDs[t.ID] {
			out = append(out, t)
		}
	}
	return out
}

// FilterAuthorOnly keeps only tweets authored by authorID.
func FilterAuthorOnly(tweets []types.TweetData, authorID string) []types.TweetData {
	var out []types.TweetData
	for _, t := range tweets {
		if t.AuthorID == authorID {
			out = append(out, t)
		}
	}
	return out
}

// FilterFullChain returns all tweets in the thread regardless of author.
func FilterFullChain(tweets []types.TweetData) []types.TweetData {
	return tweets
}
