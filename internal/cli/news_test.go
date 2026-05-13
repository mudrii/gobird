package cli

import (
	"slices"
	"testing"
)

func TestBuildNewsOpts(t *testing.T) {
	defaults := []string{"forYou", "news"}
	tests := []struct {
		name        string
		tabsFlag    string
		limit       int
		defaultTabs []string
		includeRaw  bool
		wantTabs    []string
		wantMax     int
		wantRaw     bool
	}{
		{"empty uses defaults", "", 0, defaults, false, defaults, 20, false},
		{"single tab", "forYou", 10, defaults, false, []string{"forYou"}, 10, false},
		{"multi tabs", "forYou,news", 5, defaults, false, []string{"forYou", "news"}, 5, false},
		{"whitespace trimmed", "  forYou , news  ", 0, defaults, false, []string{"forYou", "news"}, 20, false},
		{"empty entries dropped", ",,foo,,", 3, defaults, false, []string{"foo"}, 3, false},
		{"nil defaults remain nil when flag empty", "", 0, nil, false, nil, 20, false},
		{"include raw passes through", "x", 1, defaults, true, []string{"x"}, 1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildNewsOpts(tt.tabsFlag, tt.limit, tt.defaultTabs, tt.includeRaw)
			if got == nil {
				t.Fatal("buildNewsOpts returned nil")
			}
			if !slices.Equal(got.Tabs, tt.wantTabs) {
				t.Errorf("Tabs: got %v, want %v", got.Tabs, tt.wantTabs)
			}
			if got.MaxCount != tt.wantMax {
				t.Errorf("MaxCount: got %d, want %d", got.MaxCount, tt.wantMax)
			}
			if got.IncludeRaw != tt.wantRaw {
				t.Errorf("IncludeRaw: got %v, want %v", got.IncludeRaw, tt.wantRaw)
			}
		})
	}
}
