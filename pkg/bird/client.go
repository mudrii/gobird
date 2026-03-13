package bird

import (
	"github.com/mudrii/gobird/internal/client"
)

// Client is the public Twitter/X API client.
type Client struct {
	c *client.Client
}

// ClientOptions configures the Client.
type ClientOptions = client.Options

// New creates a Client from resolved credentials.
func New(creds *TwitterCookies, opts *ClientOptions) (*Client, error) {
	if creds == nil || creds.AuthToken == "" || creds.Ct0 == "" {
		return nil, errMissingCredentials
	}
	return &Client{c: client.New(creds.AuthToken, creds.Ct0, opts)}, nil
}

// NewWithTokens creates a Client from bare token strings.
func NewWithTokens(authToken, ct0 string, opts *ClientOptions) (*Client, error) {
	if authToken == "" || ct0 == "" {
		return nil, errMissingCredentials
	}
	return &Client{c: client.New(authToken, ct0, opts)}, nil
}
