package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// FormatJSON writes v as JSON to w. If pretty is true, output is indented.
func FormatJSON(w io.Writer, v any, pretty bool) error {
	enc := json.NewEncoder(w)
	if pretty {
		enc.SetIndent("", "  ")
	}
	return enc.Encode(v)
}

// FormatJSONL writes each item in items as a separate JSON line to w.
func FormatJSONL(w io.Writer, items []any) error {
	enc := json.NewEncoder(w)
	for _, item := range items {
		if err := enc.Encode(item); err != nil {
			return err
		}
	}
	return nil
}

// FormatQuiet extracts a single top-level field from v (which must be a
// map[string]any) and writes its value as a plain string to w.
func FormatQuiet(w io.Writer, v any, field string) error {
	m, ok := v.(map[string]any)
	if !ok {
		return fmt.Errorf("value is not an object")
	}
	val, exists := m[field]
	if !exists {
		return fmt.Errorf("field %q not found", field)
	}
	_, err := fmt.Fprintln(w, val)
	return err
}

// FormatField extracts a dot-delimited path from v and writes the value to w.
// Example path: "user.name" extracts v["user"]["name"].
func FormatField(w io.Writer, v any, path string) error {
	parts := strings.Split(path, ".")
	current := v
	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return fmt.Errorf("cannot traverse path %q: not an object at %q", path, part)
		}
		val, exists := m[part]
		if !exists {
			return fmt.Errorf("field %q not found in path %q", part, path)
		}
		current = val
	}
	_, err := fmt.Fprintln(w, current)
	return err
}

// FormatTable writes rows as an aligned table to w with the given headers.
// Headers are printed in uppercase. Rows is a slice of string maps.
func FormatTable(w io.Writer, rows []map[string]string, headers []string) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	// Header row.
	headerUpper := make([]string, len(headers))
	for i, h := range headers {
		headerUpper[i] = strings.ToUpper(h)
	}
	fmt.Fprintln(tw, strings.Join(headerUpper, "\t"))

	// Data rows.
	for _, row := range rows {
		cols := make([]string, len(headers))
		for i, h := range headers {
			cols[i] = row[h]
		}
		fmt.Fprintln(tw, strings.Join(cols, "\t"))
	}

	return tw.Flush()
}
