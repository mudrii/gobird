package client

import (
	"testing"
)

func TestBaseHeaders_allFieldsSet(t *testing.T) {
	h := baseHeaders("myAuth", "myCt0", "myUUID", "myDevice", "myUser")

	tests := []struct {
		header string
		want   string
	}{
		{"accept", "*/*"},
		{"accept-language", "en-US,en;q=0.9"},
		{"authorization", "Bearer " + BearerToken},
		{"x-twitter-auth-type", "OAuth2Session"},
		{"x-twitter-active-user", "yes"},
		{"x-twitter-client-language", "en"},
		{"x-csrf-token", "myCt0"},
		{"x-client-uuid", "myUUID"},
		{"x-twitter-client-deviceid", "myDevice"},
		{"x-twitter-client-user-id", "myUser"},
		{"cookie", "auth_token=myAuth; ct0=myCt0"},
		{"user-agent", UserAgent},
		{"origin", "https://x.com"},
		{"referer", "https://x.com/"},
	}
	for _, tt := range tests {
		got := h.Get(tt.header)
		if got != tt.want {
			t.Errorf("header %q: want %q, got %q", tt.header, tt.want, got)
		}
	}
}

func TestBaseHeaders_emptyClientUUID(t *testing.T) {
	h := baseHeaders("a", "b", "", "dev", "")
	if h.Get("x-client-uuid") != "" {
		t.Error("x-client-uuid should be omitted when clientUUID is empty")
	}
}

func TestBaseHeaders_emptyDeviceID(t *testing.T) {
	h := baseHeaders("a", "b", "uuid", "", "")
	if h.Get("x-twitter-client-deviceid") != "" {
		t.Error("x-twitter-client-deviceid should be omitted when deviceID is empty")
	}
}

func TestBaseHeaders_emptyUserID(t *testing.T) {
	h := baseHeaders("a", "b", "uuid", "dev", "")
	if h.Get("x-twitter-client-user-id") != "" {
		t.Error("x-twitter-client-user-id should be omitted when userID is empty")
	}
}

func TestBaseHeaders_transactionIDPresent(t *testing.T) {
	h := baseHeaders("a", "b", "uuid", "dev", "")
	txID := h.Get("x-client-transaction-id")
	if txID == "" {
		t.Error("x-client-transaction-id should be set")
	}
	if len(txID) != 32 {
		t.Errorf("x-client-transaction-id should be 32 hex chars, got %d: %q", len(txID), txID)
	}
}

func TestBaseHeaders_transactionIDUnique(t *testing.T) {
	h1 := baseHeaders("a", "b", "", "", "")
	h2 := baseHeaders("a", "b", "", "", "")
	if h1.Get("x-client-transaction-id") == h2.Get("x-client-transaction-id") {
		t.Error("transaction IDs should be unique across calls")
	}
}

func TestJsonHeaders_hasContentType(t *testing.T) {
	h := jsonHeaders("a", "b", "uuid", "dev", "")
	ct := h.Get("content-type")
	if ct != "application/json" {
		t.Errorf("content-type: want application/json, got %q", ct)
	}
}

func TestJsonHeaders_includesBaseHeaders(t *testing.T) {
	h := jsonHeaders("a", "b", "uuid", "dev", "")
	if h.Get("authorization") == "" {
		t.Error("jsonHeaders should include base header authorization")
	}
	if h.Get("x-csrf-token") != "b" {
		t.Errorf("x-csrf-token: want b, got %q", h.Get("x-csrf-token"))
	}
}

func TestUploadHeaders_noContentType(t *testing.T) {
	h := uploadHeaders("a", "b", "uuid", "dev", "")
	ct := h.Get("content-type")
	if ct != "" {
		t.Errorf("uploadHeaders should not set content-type, got %q", ct)
	}
}

func TestUploadHeaders_includesBaseHeaders(t *testing.T) {
	h := uploadHeaders("a", "b", "uuid", "dev", "")
	if h.Get("authorization") == "" {
		t.Error("uploadHeaders should include base header authorization")
	}
}

func TestCreateTransactionID_length(t *testing.T) {
	id := createTransactionID()
	if len(id) != 32 {
		t.Errorf("createTransactionID: want 32 hex chars, got %d: %q", len(id), id)
	}
}

func TestCreateTransactionID_hexChars(t *testing.T) {
	id := createTransactionID()
	for _, c := range id {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			t.Errorf("createTransactionID: non-hex char %q in %q", string(c), id)
			break
		}
	}
}

func TestClientGetJSONHeaders(t *testing.T) {
	c := New("tok", "csrf", nil)
	h := c.getJSONHeaders()
	if h.Get("content-type") != "application/json" {
		t.Error("getJSONHeaders should set content-type to application/json")
	}
	if h.Get("x-csrf-token") != "csrf" {
		t.Errorf("x-csrf-token: want csrf, got %q", h.Get("x-csrf-token"))
	}
}

func TestClientGetBaseHeaders(t *testing.T) {
	c := New("tok", "csrf", nil)
	h := c.getBaseHeaders()
	if h.Get("content-type") != "" {
		t.Error("getBaseHeaders should not set content-type")
	}
	if h.Get("x-csrf-token") != "csrf" {
		t.Errorf("x-csrf-token: want csrf, got %q", h.Get("x-csrf-token"))
	}
}

func TestClientGetUploadHeaders(t *testing.T) {
	c := New("tok", "csrf", nil)
	h := c.getUploadHeaders()
	if h.Get("content-type") != "" {
		t.Error("getUploadHeaders should not set content-type")
	}
}

func TestBaseHeaders_allEmpty(t *testing.T) {
	h := baseHeaders("", "", "", "", "")
	if h.Get("cookie") != "auth_token=; ct0=" {
		t.Errorf("cookie: got %q", h.Get("cookie"))
	}
	if h.Get("x-csrf-token") != "" {
		t.Errorf("x-csrf-token should be empty string, got %q", h.Get("x-csrf-token"))
	}
}
