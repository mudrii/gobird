package cli

import (
	"testing"

	"github.com/mudrii/gobird/internal/config"
)

func TestValidateOutputFlags(t *testing.T) {
	cases := []struct {
		name    string
		json    bool
		jsonF   bool
		plain   bool
		wantErr bool
	}{
		{"none set", false, false, false, false},
		{"json only", true, false, false, false},
		{"json-full only", false, true, false, false},
		{"plain only", false, false, true, false},
		{"json and plain", true, false, true, true},
		{"json and json-full", true, true, false, true},
		{"json-full and plain", false, true, true, true},
		{"all three", true, true, true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			old := globalFlags
			defer func() { globalFlags = old }()
			globalFlags.jsonOutput = tc.json
			globalFlags.jsonFull = tc.jsonF
			globalFlags.plain = tc.plain
			err := validateOutputFlags()
			if (err != nil) != tc.wantErr {
				t.Errorf("validateOutputFlags: err=%v, wantErr=%v", err, tc.wantErr)
			}
		})
	}
}

func TestValidateLimit(t *testing.T) {
	cases := []struct {
		limit   int
		wantErr bool
	}{
		{0, false},
		{1, false},
		{100, false},
		{-1, true},
		{-100, true},
	}
	for _, tc := range cases {
		err := validateLimit(tc.limit)
		if (err != nil) != tc.wantErr {
			t.Errorf("validateLimit(%d): err=%v, wantErr=%v", tc.limit, err, tc.wantErr)
		}
	}
}

func TestValidateMediaMIME(t *testing.T) {
	cases := []struct {
		mime    string
		wantErr bool
	}{
		{"image/jpeg", false},
		{"image/png", false},
		{"image/gif", false},
		{"video/mp4", false},
		{"audio/mpeg", false},
		{"application/pdf", true},
		{"text/plain", true},
		{"application/octet-stream", true},
		{"", true},
	}
	for _, tc := range cases {
		err := validateMediaMIME(tc.mime)
		if (err != nil) != tc.wantErr {
			t.Errorf("validateMediaMIME(%q): err=%v, wantErr=%v", tc.mime, err, tc.wantErr)
		}
	}
}

func TestFirstNonEmptyString(t *testing.T) {
	cases := []struct {
		vals []string
		want string
	}{
		{[]string{"a", "b"}, "a"},
		{[]string{"", "b"}, "b"},
		{[]string{"", "", "c"}, "c"},
		{[]string{"", ""}, ""},
		{[]string{"  ", "b"}, "b"},
		{nil, ""},
	}
	for _, tc := range cases {
		got := firstNonEmptyString(tc.vals...)
		if got != tc.want {
			t.Errorf("firstNonEmptyString(%v) = %q, want %q", tc.vals, got, tc.want)
		}
	}
}

func TestResolveTimeoutMs(t *testing.T) {
	cases := []struct {
		name      string
		flagVal   int
		configVal int
		want      int
	}{
		{"flag wins", 5000, 1000, 5000},
		{"config fallback", 0, 2000, 2000},
		{"both zero", 0, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			old := globalFlags.timeoutMs
			defer func() { globalFlags.timeoutMs = old }()
			globalFlags.timeoutMs = tc.flagVal
			cfg := &config.Config{TimeoutMs: tc.configVal}
			got := resolveTimeoutMs(cfg)
			if got != tc.want {
				t.Errorf("resolveTimeoutMs: want %d, got %d", tc.want, got)
			}
		})
	}
}

func TestResolveCookieTimeoutMs(t *testing.T) {
	cases := []struct {
		name      string
		flagVal   int
		configVal int
		want      int
	}{
		{"flag wins", 3000, 1000, 3000},
		{"config fallback", 0, 4000, 4000},
		{"both zero", 0, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			old := globalFlags.cookieTimeoutMs
			defer func() { globalFlags.cookieTimeoutMs = old }()
			globalFlags.cookieTimeoutMs = tc.flagVal
			cfg := &config.Config{CookieTimeoutMs: tc.configVal}
			got := resolveCookieTimeoutMs(cfg)
			if got != tc.want {
				t.Errorf("resolveCookieTimeoutMs: want %d, got %d", tc.want, got)
			}
		})
	}
}

func TestResolveQuoteDepth_ConfigFallback(t *testing.T) {
	old := globalFlags.quoteDepth
	defer func() { globalFlags.quoteDepth = old }()

	globalFlags.quoteDepth = -1
	five := 5
	cfg := &config.Config{QuoteDepth: &five}
	got := resolveQuoteDepth(cfg)
	if got != 5 {
		t.Errorf("resolveQuoteDepth: want 5, got %d", got)
	}
}

func TestResolveQuoteDepth_FlagZero(t *testing.T) {
	old := globalFlags.quoteDepth
	defer func() { globalFlags.quoteDepth = old }()

	globalFlags.quoteDepth = 0
	five := 5
	cfg := &config.Config{QuoteDepth: &five}
	got := resolveQuoteDepth(cfg)
	if got != 0 {
		t.Errorf("resolveQuoteDepth: want 0, got %d", got)
	}
}

func TestStripAtPrefix(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"@user", "user"},
		{"user", "user"},
		{"@", ""},
		{"", ""},
		{"@@double", "@double"},
	}
	for _, tc := range cases {
		got := stripAtPrefix(tc.input)
		if got != tc.want {
			t.Errorf("stripAtPrefix(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestLoadMedia_NonexistentFile(t *testing.T) {
	_, err := loadMedia("/nonexistent/file.jpg")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestResolveCookieSources(t *testing.T) {
	cases := []struct {
		name          string
		flagSources   []string
		flagBrowser   string
		cfgSource     []string
		cfgBrowser    string
		wantLen       int
		wantFirstElem string
	}{
		{"flag sources win", []string{"chrome"}, "", nil, "", 1, "chrome"},
		{"config source fallback", nil, "", []string{"firefox"}, "", 1, "firefox"},
		{"flag browser fallback", nil, "safari", nil, "", 1, "safari"},
		{"config browser fallback", nil, "", nil, "chrome", 1, "chrome"},
		{"all empty", nil, "", nil, "", 0, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			old := globalFlags
			defer func() { globalFlags = old }()
			globalFlags.cookieSources = tc.flagSources
			globalFlags.browser = tc.flagBrowser
			cfg := &config.Config{
				CookieSource:   config.StringOrSlice(tc.cfgSource),
				DefaultBrowser: tc.cfgBrowser,
			}
			got := resolveCookieSources(cfg)
			if len(got) != tc.wantLen {
				t.Errorf("len: want %d, got %d (%v)", tc.wantLen, len(got), got)
			}
			if tc.wantLen > 0 && got[0] != tc.wantFirstElem {
				t.Errorf("first elem: want %q, got %q", tc.wantFirstElem, got[0])
			}
		})
	}
}

func TestResolveChromeProfile(t *testing.T) {
	old := globalFlags
	defer func() { globalFlags = old }()

	globalFlags.chromeProfileDir = ""
	globalFlags.chromeProfile = "flagProfile"
	cfg := &config.Config{ChromeProfileDir: "configDir", ChromeProfile: "configProfile"}
	got := resolveChromeProfile(cfg)
	if got != "flagProfile" {
		t.Errorf("resolveChromeProfile: want %q, got %q", "flagProfile", got)
	}

	globalFlags.chromeProfileDir = "/custom/dir"
	got = resolveChromeProfile(cfg)
	if got != "/custom/dir" {
		t.Errorf("resolveChromeProfile with dir: want %q, got %q", "/custom/dir", got)
	}
}
