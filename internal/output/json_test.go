package output_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/mudrii/gobird/internal/output"
)

func TestToJSON_ValidStruct(t *testing.T) {
	v := map[string]string{"key": "value"}
	b, err := output.ToJSON(v)
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}
	if !json.Valid(b) {
		t.Errorf("ToJSON produced invalid JSON: %s", b)
	}
	if !strings.Contains(string(b), "\"key\"") {
		t.Errorf("ToJSON output missing key: %s", b)
	}
	if !strings.Contains(string(b), "\"value\"") {
		t.Errorf("ToJSON output missing value: %s", b)
	}
}

func TestToJSON_Indented(t *testing.T) {
	v := map[string]int{"count": 3}
	b, err := output.ToJSON(v)
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, "\n") || !strings.Contains(s, "  ") {
		t.Errorf("ToJSON output does not appear indented: %s", s)
	}
}

func TestToJSON_WithOmitempty(t *testing.T) {
	type row struct {
		Name  string `json:"name"`
		Score int    `json:"score,omitempty"`
	}
	v := row{Name: "alice"}
	b, err := output.ToJSON(v)
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}
	s := string(b)
	if strings.Contains(s, "score") {
		t.Errorf("omitempty zero field should be absent: %s", s)
	}
}

func TestPrintJSON_WritesToWriter(t *testing.T) {
	var buf bytes.Buffer
	v := map[string]bool{"ok": true}
	if err := output.PrintJSON(&buf, v); err != nil {
		t.Fatalf("PrintJSON: %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, "\"ok\"") {
		t.Errorf("PrintJSON output missing expected key: %q", got)
	}
	if !strings.HasSuffix(got, "\n") {
		t.Errorf("PrintJSON output should end with newline: %q", got)
	}
}

func TestPrintJSON_EmptySlice(t *testing.T) {
	var buf bytes.Buffer
	if err := output.PrintJSON(&buf, []string{}); err != nil {
		t.Fatalf("PrintJSON empty slice: %v", err)
	}
	got := strings.TrimSpace(buf.String())
	if got != "[]" {
		t.Errorf("expected '[]', got %q", got)
	}
}

func TestPrintJSON_NilValue(t *testing.T) {
	var buf bytes.Buffer
	if err := output.PrintJSON(&buf, nil); err != nil {
		t.Fatalf("PrintJSON nil: %v", err)
	}
	got := strings.TrimSpace(buf.String())
	if got != "null" {
		t.Errorf("expected 'null', got %q", got)
	}
}

// errorWriter is an io.Writer that always returns an error.
type errorWriter struct{}

func (errorWriter) Write(_ []byte) (int, error) {
	return 0, fmt.Errorf("simulated write error")
}

func TestPrintJSON_PropagatesWriteError(t *testing.T) {
	err := output.PrintJSON(errorWriter{}, map[string]string{"k": "v"})
	if err == nil {
		t.Fatal("expected error when writer fails, got nil")
	}
	if !strings.Contains(err.Error(), "simulated write error") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestToJSON_NilInput(t *testing.T) {
	b, err := output.ToJSON(nil)
	if err != nil {
		t.Fatalf("ToJSON(nil): %v", err)
	}
	if string(b) != "null" {
		t.Errorf("expected 'null', got %q", string(b))
	}
}
