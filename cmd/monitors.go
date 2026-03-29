package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/larmhq/larm-cli/internal/client"
	"github.com/larmhq/larm-cli/internal/output"
)

var monitorsCmd = &cobra.Command{
	Use:   "monitors",
	Short: "Manage uptime monitors",
}

var monitorsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List monitors",
	Example: `  larm monitors list
  larm monitors list --check-type http
  larm monitors list --output json --jq '.[].name'`,
	RunE: runMonitorsList,
}

var monitorsShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a monitor",
	Example: `  larm monitors show 5942d5c4-6bb0-4d7a-b743-872c32a777ae
  larm monitors show <id> --output json --jq '.current_state'`,
	Args: cobra.ExactArgs(1),
	RunE: runMonitorsShow,
}

var monitorsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a monitor",
	Example: `  larm monitors create --name "API" --url https://example.com
  larm monitors create --name "TCP" --check-type tcp --config '{"host":"db.example.com","port":5432}'`,
	RunE: runMonitorsCreate,
}

var monitorsUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a monitor",
	Example: `  larm monitors update <id> --name "New Name"
  larm monitors update <id> --enabled false`,
	Args: cobra.ExactArgs(1),
	RunE: runMonitorsUpdate,
}

var monitorsDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a monitor",
	Args:  cobra.ExactArgs(1),
	RunE:  runMonitorsDelete,
}

var monitorsStateCmd = &cobra.Command{
	Use:   "state <id>",
	Short: "Get current monitor state",
	Args:  cobra.ExactArgs(1),
	RunE:  runMonitorsState,
}

var monitorsUptimeCmd = &cobra.Command{
	Use:   "uptime <id>",
	Short: "Get monitor uptime",
	Args:  cobra.ExactArgs(1),
	RunE:  runMonitorsUptime,
}

var monitorsResponseTimesCmd = &cobra.Command{
	Use:   "response-times <id>",
	Short: "Get monitor response times",
	Args:  cobra.ExactArgs(1),
	RunE:  runMonitorsResponseTimes,
}

var monitorsCertCmd = &cobra.Command{
	Use:   "cert <id>",
	Short: "Get SSL certificate info",
	Args:  cobra.ExactArgs(1),
	RunE:  runMonitorsCert,
}

func init() {
	monitorsListCmd.Flags().String("check-type", "", "Filter by check type (http, tcp, dns, heartbeat, synthetic)")
	monitorsListCmd.Flags().String("enabled", "", "Filter by enabled status (true, false)")

	monitorsCreateCmd.Flags().String("name", "", "Monitor name")
	monitorsCreateCmd.Flags().String("check-type", "http", "Check type (http, tcp, dns, heartbeat, synthetic)")
	monitorsCreateCmd.Flags().String("url", "", "URL to monitor (http)")
	monitorsCreateCmd.Flags().Int("interval", 180, "Check interval in seconds")
	monitorsCreateCmd.Flags().Int("timeout", 10000, "Timeout in milliseconds")
	monitorsCreateCmd.Flags().String("config", "", "Full config as JSON (advanced)")
	_ = monitorsCreateCmd.MarkFlagRequired("name")

	monitorsUpdateCmd.Flags().String("name", "", "New monitor name")
	monitorsUpdateCmd.Flags().String("enabled", "", "Enable or disable (true, false)")
	monitorsUpdateCmd.Flags().Int("interval", 0, "New check interval in seconds")
	monitorsUpdateCmd.Flags().Int("timeout", 0, "New timeout in milliseconds")

	monitorsUptimeCmd.Flags().String("range", "", "Time range (e.g. 24h, 7d, 30d)")

	monitorsCmd.AddCommand(monitorsListCmd)
	monitorsCmd.AddCommand(monitorsShowCmd)
	monitorsCmd.AddCommand(monitorsCreateCmd)
	monitorsCmd.AddCommand(monitorsUpdateCmd)
	monitorsCmd.AddCommand(monitorsDeleteCmd)
	monitorsCmd.AddCommand(monitorsStateCmd)
	monitorsCmd.AddCommand(monitorsUptimeCmd)
	monitorsCmd.AddCommand(monitorsResponseTimesCmd)
	monitorsCmd.AddCommand(monitorsCertCmd)
	rootCmd.AddCommand(monitorsCmd)
}

func runMonitorsList(cmd *cobra.Command, _ []string) error {
	c, err := newTypedClient(cmd)
	if err != nil {
		return err
	}

	params := &client.ListMonitorsParams{}

	if ct, _ := cmd.Flags().GetString("check-type"); ct != "" {
		checkType := client.CheckType(ct)
		params.CheckType = &checkType
	}
	if en, _ := cmd.Flags().GetString("enabled"); en != "" {
		enabled := en == "true"
		params.Enabled = &enabled
	}

	resp, err := c.ListMonitorsWithResponse(cmd.Context(), params)
	if err != nil {
		return err
	}

	return handleAndPrintWithDefaults(cmd, resp.StatusCode(), resp.Body,
		"id,name,check_type,current_state,enabled",
		output.PrintOpts{ColorHints: map[string]output.ColorFunc{
			"current_state": output.StatusColor,
			"enabled":       output.BoolColor,
		}})
}

func runMonitorsShow(cmd *cobra.Command, args []string) error {
	c, err := newTypedClient(cmd)
	if err != nil {
		return err
	}

	id, err := parseUUID(args[0])
	if err != nil {
		return err
	}

	resp, err := c.GetMonitorWithResponse(cmd.Context(), id)
	if err != nil {
		return err
	}

	return handleAndPrintWithDefaults(cmd, resp.StatusCode(), resp.Body, "",
		output.PrintOpts{ViewFunc: viewMonitor})
}

func runMonitorsCreate(cmd *cobra.Command, _ []string) error {
	c, err := newTypedClient(cmd)
	if err != nil {
		return err
	}

	name, _ := cmd.Flags().GetString("name")
	checkType, _ := cmd.Flags().GetString("check-type")
	url, _ := cmd.Flags().GetString("url")
	interval, _ := cmd.Flags().GetInt("interval")
	timeout, _ := cmd.Flags().GetInt("timeout")
	configJSON, _ := cmd.Flags().GetString("config")

	ct := client.CheckType(checkType)
	body := client.CreateMonitorJSONRequestBody{
		Name:            &name,
		CheckType:       &ct,
		IntervalSeconds: &interval,
		TimeoutMs:       &timeout,
	}

	if configJSON != "" {
		var cfg map[string]interface{}
		if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
			return fmt.Errorf("invalid --config JSON: %w", err)
		}
		body.Config = &cfg
	} else if url != "" {
		cfg := map[string]interface{}{
			"url":                   url,
			"method":                "GET",
			"expected_status_codes": []int{200},
			"follow_redirects":      true,
		}
		body.Config = &cfg
	}

	resp, err := c.CreateMonitorWithResponse(cmd.Context(), body)
	if err != nil {
		return err
	}

	return handleAndPrintWithDefaults(cmd, resp.StatusCode(), resp.Body, "",
		output.PrintOpts{ViewFunc: viewMonitor})
}

func runMonitorsUpdate(cmd *cobra.Command, args []string) error {
	c, err := newTypedClient(cmd)
	if err != nil {
		return err
	}

	id, err := parseUUID(args[0])
	if err != nil {
		return err
	}

	body := client.UpdateMonitorJSONRequestBody{}

	if cmd.Flags().Changed("name") {
		name, _ := cmd.Flags().GetString("name")
		body.Name = &name
	}
	if cmd.Flags().Changed("enabled") {
		en, _ := cmd.Flags().GetString("enabled")
		enabled := en == "true"
		body.Enabled = &enabled
	}
	if cmd.Flags().Changed("interval") {
		interval, _ := cmd.Flags().GetInt("interval")
		body.IntervalSeconds = &interval
	}
	if cmd.Flags().Changed("timeout") {
		timeout, _ := cmd.Flags().GetInt("timeout")
		body.TimeoutMs = &timeout
	}

	resp, err := c.UpdateMonitorWithResponse(cmd.Context(), id, body)
	if err != nil {
		return err
	}

	return handleAndPrintWithDefaults(cmd, resp.StatusCode(), resp.Body, "",
		output.PrintOpts{ViewFunc: viewMonitor})
}

func runMonitorsDelete(cmd *cobra.Command, args []string) error {
	if err := confirmAction(cmd, fmt.Sprintf("Delete monitor %s?", args[0])); err != nil {
		return err
	}

	c, err := newTypedClient(cmd)
	if err != nil {
		return err
	}

	id, err := parseUUID(args[0])
	if err != nil {
		return err
	}

	resp, err := c.DeleteMonitorWithResponse(cmd.Context(), id)
	if err != nil {
		return err
	}

	return handleDelete(cmd, resp.StatusCode(), resp.Body)
}

func runMonitorsState(cmd *cobra.Command, args []string) error {
	c, err := newTypedClient(cmd)
	if err != nil {
		return err
	}

	id, err := parseUUID(args[0])
	if err != nil {
		return err
	}

	resp, err := c.GetMonitorStateWithResponse(cmd.Context(), id)
	if err != nil {
		return err
	}

	return handleAndPrintWithDefaults(cmd, resp.StatusCode(), resp.Body, "",
		output.PrintOpts{ViewFunc: viewState})
}

func runMonitorsUptime(cmd *cobra.Command, args []string) error {
	c, err := newTypedClient(cmd)
	if err != nil {
		return err
	}

	id, err := parseUUID(args[0])
	if err != nil {
		return err
	}

	params := &client.GetMonitorUptimeParams{}
	if r, _ := cmd.Flags().GetString("range"); r != "" {
		params.Range = &r
	}

	resp, err := c.GetMonitorUptimeWithResponse(cmd.Context(), id, params)
	if err != nil {
		return err
	}

	return handleAndPrintWithDefaults(cmd, resp.StatusCode(), resp.Body, "",
		output.PrintOpts{ViewFunc: viewUptime})
}

func runMonitorsResponseTimes(cmd *cobra.Command, args []string) error {
	c, err := newTypedClient(cmd)
	if err != nil {
		return err
	}

	id, err := parseUUID(args[0])
	if err != nil {
		return err
	}

	resp, err := c.GetMonitorResponseTimesWithResponse(cmd.Context(), id)
	if err != nil {
		return err
	}

	return handleAndPrintWithDefaults(cmd, resp.StatusCode(), resp.Body, "",
		output.PrintOpts{ViewFunc: viewResponseTimes})
}

func runMonitorsCert(cmd *cobra.Command, args []string) error {
	c, err := newTypedClient(cmd)
	if err != nil {
		return err
	}

	id, err := parseUUID(args[0])
	if err != nil {
		return err
	}

	resp, err := c.GetMonitorCertWithResponse(cmd.Context(), id)
	if err != nil {
		return err
	}

	return handleAndPrintWithDefaults(cmd, resp.StatusCode(), resp.Body, "",
		output.PrintOpts{ViewFunc: viewCert})
}

func viewMonitor(w io.Writer, data json.RawMessage) error {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	// Extract config fields
	var configRows []viewRow
	if cfg, ok := m["config"].(map[string]any); ok {
		for _, k := range []string{"url", "host", "port", "method", "expected_status_codes", "follow_redirects", "expected_keyword", "unexpected_keyword", "record_type", "nameserver", "expected_records", "expected_interval", "consecutive_misses"} {
			if v, ok := cfg[k]; ok && v != nil {
				configRows = append(configRows, staticField(k, joinStrings(v, fmt.Sprintf("%v", v))))
			}
		}
	}

	rows := []viewRow{
		field("id", "id"),
		field("name", "name"),
		field("check_type", "check_type"),
		colorField("current_state", "current_state", output.StatusColor),
		colorField("enabled", "enabled", output.BoolColor),
		field("interval_seconds", "interval_seconds"),
		field("timeout_ms", "timeout_ms"),
		field("confirm_down_minutes", "confirm_down_minutes"),
		field("confirm_up_minutes", "confirm_up_minutes"),
	}
	rows = append(rows, configRows...)
	rows = append(rows,
		staticField("alert_channels", joinStrings(m["alert_channel_ids"], "(none)")),
		field("inserted_at", "inserted_at"),
		field("updated_at", "updated_at"),
	)

	return writeView(w, m, rows)
}

func viewState(w io.Writer, data json.RawMessage) error {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	return writeView(w, m, []viewRow{
		colorField("state", "state", output.StatusColor),
		field("entered_at", "entered_at"),
	})
}

func viewUptime(w io.Writer, data json.RawMessage) error {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	rows := []viewRow{}
	if pct, ok := m["uptime_pct"].(float64); ok {
		rows = append(rows, staticField("uptime", fmt.Sprintf("%.2f%%", pct)))
	}
	rows = append(rows, separator())

	if dist, ok := m["distribution"].(map[string]any); ok {
		for _, k := range []string{"pass", "fail", "error", "timeout", "total"} {
			if v, ok := dist[k]; ok {
				rows = append(rows, staticField(k, fmt.Sprintf("%v", v)))
			}
		}
	}

	return writeView(w, m, rows)
}

func viewResponseTimes(w io.Writer, data json.RawMessage) error {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	rows := []viewRow{}

	if p95, ok := m["p95"].(float64); ok {
		rows = append(rows, staticField("p95", fmt.Sprintf("%.1fms", p95)))
	}

	if locs, ok := m["by_location"].([]any); ok && len(locs) > 0 {
		rows = append(rows, separator())
		for _, loc := range locs {
			if l, ok := loc.(map[string]any); ok {
				name := fmt.Sprintf("%v", l["location"])
				if p95, ok := l["p95"].(float64); ok {
					rows = append(rows, staticField(name, fmt.Sprintf("%.1fms (%v checks)", p95, l["checks"])))
				}
			}
		}
	}

	return writeView(w, m, rows)
}

func viewCert(w io.Writer, data json.RawMessage) error {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	return writeView(w, m, []viewRow{
		colorField("cert_valid", "cert_valid", output.BoolColor),
		field("cert_subject", "cert_subject"),
		field("cert_issuer", "cert_issuer"),
		field("cert_expiry", "cert_expiry"),
		field("days_remaining", "days_remaining"),
	})
}

func parseUUID(s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("invalid ID: %s", s)
	}
	return id, nil
}
