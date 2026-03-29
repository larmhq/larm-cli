package cmd

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/larmhq/larm-cli/internal/output"
)

type viewRow struct {
	label string
	key   string           // dot-path into JSON; empty if value is pre-computed
	value string           // pre-computed value; used when key is empty
	color output.ColorFunc // optional color function
}

func field(label, key string) viewRow {
	return viewRow{label: label, key: key}
}

func colorField(label, key string, color output.ColorFunc) viewRow {
	return viewRow{label: label, key: key, color: color}
}

func staticField(label, value string) viewRow {
	return viewRow{label: label, value: value}
}

func separator() viewRow {
	return viewRow{}
}

// writeView renders a single object as key-value pairs with controlled field order.
// Empty/nil values are skipped. Timestamps are humanized in TTY mode.
func writeView(w io.Writer, obj map[string]any, rows []viewRow) error {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)

	for _, row := range rows {
		// Separator
		if row.label == "" && row.key == "" && row.value == "" {
			fmt.Fprintln(tw)
			continue
		}

		val := row.value
		if val == "" && row.key != "" {
			val = output.ResolveField(obj, row.key)
		}

		if val == "" {
			continue
		}

		if row.color != nil {
			val = row.color(val)
		}

		fmt.Fprintf(tw, "%s:\t%s\n", row.label, val)
	}

	return tw.Flush()
}

// joinStrings extracts string values from an []any and joins them.
func joinStrings(v any, fallback string) string {
	arr, ok := v.([]any)
	if !ok || len(arr) == 0 {
		return fallback
	}
	parts := make([]string, len(arr))
	for i, item := range arr {
		if m, ok := item.(map[string]any); ok {
			if name, ok := m["name"].(string); ok {
				parts[i] = name
				continue
			}
		}
		parts[i] = fmt.Sprintf("%v", item)
	}
	return strings.Join(parts, ", ")
}
