package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/itchyny/gojq"
	"github.com/mattn/go-isatty"
)

// IsTTY returns true if stdout is a terminal.
func IsTTY() bool {
	return isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
}

// ColorFunc maps a string value to a colorized string.
type ColorFunc func(string) string

// PrintOpts configures optional behavior for Print.
type PrintOpts struct {
	// ColorHints maps column names to color functions.
	// Only applied in table/key-value mode when the terminal supports color.
	ColorHints map[string]ColorFunc
	// ViewFunc renders a single object for human display. Called instead of the
	// generic table renderer when set and format is "table". JSON mode bypasses it.
	ViewFunc func(io.Writer, json.RawMessage) error
}

// Print writes data to stdout in the requested format.
func Print(w io.Writer, format, jqExpr, fields string, data json.RawMessage, opts ...PrintOpts) error {
	switch format {
	case "json":
		return printJSON(w, jqExpr, fields, data)
	case "table":
		var o PrintOpts
		if len(opts) > 0 {
			o = opts[0]
		}
		if o.ViewFunc != nil {
			return o.ViewFunc(w, data)
		}
		return printTable(w, fields, data, o)
	default:
		return fmt.Errorf("unknown output format: %s", format)
	}
}

// StatusColor colors known status values (up/down/stale/pending/resolved etc).
func StatusColor(v string) string {
	switch v {
	case "up", "resolved", "completed":
		return green(v)
	case "down", "critical":
		return red(v)
	case "stale", "degraded", "major":
		return yellow(v)
	case "pending", "investigating", "identified", "monitoring", "scheduled", "in_progress":
		return dim(v)
	default:
		return v
	}
}

// BoolColor colors true green and false red.
func BoolColor(v string) string {
	switch v {
	case "true":
		return green(v)
	case "false":
		return red(v)
	default:
		return v
	}
}

func shouldColor() bool {
	return IsTTY() && os.Getenv("NO_COLOR") == ""
}

func green(s string) string {
	if !shouldColor() {
		return s
	}
	return "\033[32m" + s + "\033[0m"
}

func red(s string) string {
	if !shouldColor() {
		return s
	}
	return "\033[31m" + s + "\033[0m"
}

func yellow(s string) string {
	if !shouldColor() {
		return s
	}
	return "\033[33m" + s + "\033[0m"
}

func dim(s string) string {
	if !shouldColor() {
		return s
	}
	return "\033[90m" + s + "\033[0m"
}

func printJSON(w io.Writer, jqExpr, fields string, data json.RawMessage) error {
	if fields != "" {
		var err error
		data, err = filterFields(data, strings.Split(fields, ","))
		if err != nil {
			return err
		}
	}

	if jqExpr != "" {
		return applyJQ(w, jqExpr, data)
	}

	if IsTTY() {
		var buf bytes.Buffer
		if err := json.Indent(&buf, data, "", "  "); err != nil {
			return err
		}
		_, err := fmt.Fprintln(w, buf.String())
		return err
	}

	_, err := fmt.Fprintln(w, string(data))
	return err
}

func printTable(w io.Writer, fields string, data json.RawMessage, opts PrintOpts) error {
	var items []map[string]any
	if err := json.Unmarshal(data, &items); err != nil {
		var obj map[string]any
		if err := json.Unmarshal(data, &obj); err != nil {
			_, e := fmt.Fprintln(w, string(data))
			return e
		}
		items = []map[string]any{obj}
	}

	if len(items) == 0 {
		if IsTTY() {
			fmt.Fprintln(os.Stderr, "No results found.")
		}
		return nil
	}

	var cols []string
	if fields != "" {
		for _, f := range strings.Split(fields, ",") {
			cols = append(cols, strings.TrimSpace(f))
		}
	} else {
		for k := range items[0] {
			cols = append(cols, k)
		}
		slices.Sort(cols)
	}

	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)

	// Header
	fmt.Fprintln(tw, strings.Join(cols, "\t"))

	// Rows
	for _, item := range items {
		vals := make([]string, len(cols))
		for i, col := range cols {
			v := ResolveField(item, col)
			if fn, ok := opts.ColorHints[col]; ok {
				v = fn(v)
			}
			vals[i] = v
		}
		fmt.Fprintln(tw, strings.Join(vals, "\t"))
	}

	return tw.Flush()
}

// HumanizeTimestamp converts ISO timestamps to relative time in TTY mode.
func HumanizeTimestamp(v string) string {
	if !IsTTY() {
		return v
	}

	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		t, err = time.Parse(time.RFC3339Nano, v)
		if err != nil {
			return v
		}
	}

	return relativeTime(t)
}

func relativeTime(t time.Time) string {
	d := time.Since(t)

	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", h)
	case d < 30*24*time.Hour:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("Jan 2, 2006")
	}
}

// ResolveField reads a value from a map, supporting dot-paths like "config.url".
// Humanizes timestamps in TTY mode and extracts .name from nested objects.
func ResolveField(obj map[string]any, path string) string {
	parts := strings.Split(path, ".")
	var current any = obj

	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current, ok = m[part]
		if !ok {
			return ""
		}
	}

	switch v := current.(type) {
	case nil:
		return ""
	case string:
		return HumanizeTimestamp(v)
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%g", v)
	case bool:
		return fmt.Sprintf("%t", v)
	case map[string]any:
		if name, ok := v["name"].(string); ok {
			return name
		}
		if email, ok := v["email"].(string); ok {
			return email
		}
		if id, ok := v["id"].(string); ok {
			return id
		}
		b, _ := json.Marshal(v)
		return string(b)
	case []any:
		if len(v) == 0 {
			return ""
		}
		names := make([]string, 0, len(v))
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				if name, ok := m["name"].(string); ok {
					names = append(names, name)
					continue
				}
			}
		}
		if len(names) > 0 {
			return strings.Join(names, ", ")
		}
		return fmt.Sprintf("[%d items]", len(v))
	default:
		return fmt.Sprintf("%v", v)
	}
}

func applyJQ(w io.Writer, expr string, data json.RawMessage) error {
	query, err := gojq.Parse(expr)
	if err != nil {
		return fmt.Errorf("invalid jq expression: %w", err)
	}

	var input any
	if err := json.Unmarshal(data, &input); err != nil {
		return err
	}

	iter := query.Run(input)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, isErr := v.(error); isErr {
			return fmt.Errorf("jq error: %w", err)
		}
		out, err := json.Marshal(v)
		if err != nil {
			return err
		}
		fmt.Fprintln(w, string(out))
	}
	return nil
}

func filterFields(data json.RawMessage, fields []string) (json.RawMessage, error) {
	fieldSet := make(map[string]bool, len(fields))
	for _, f := range fields {
		fieldSet[strings.TrimSpace(f)] = true
	}

	var items []map[string]any
	if json.Unmarshal(data, &items) == nil {
		for i, item := range items {
			items[i] = pickFields(item, fieldSet)
		}
		return json.Marshal(items)
	}

	var obj map[string]any
	if json.Unmarshal(data, &obj) == nil {
		return json.Marshal(pickFields(obj, fieldSet))
	}

	return data, nil
}

func pickFields(obj map[string]any, fields map[string]bool) map[string]any {
	result := make(map[string]any, len(fields))
	for k, v := range obj {
		if fields[k] {
			result[k] = v
		}
	}
	return result
}
