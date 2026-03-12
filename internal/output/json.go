package output

import (
	"encoding/json"
	"fmt"
	"io"
)

// ToJSON marshals v to indented JSON with 2-space indent.
func ToJSON(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

// PrintJSON writes v as indented JSON to w.
func PrintJSON(w io.Writer, v any) error {
	b, err := ToJSON(v)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(b))
	return err
}
