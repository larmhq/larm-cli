package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/larmhq/larm-go/client"

	"github.com/larmhq/larm-cli/internal/output"
)

var disruptionsCmd = &cobra.Command{
	Use:   "disruptions",
	Short: "Manage disruptions",
}

var disruptionsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List disruptions",
	Example: `  larm disruptions list
  larm disruptions list --output json --jq '[.[] | select(.status != "resolved")]'`,
	RunE: runDisruptionsList,
}

var disruptionsShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a disruption",
	Example: `  larm disruptions show <id>
  larm disruptions show <id> --output json --jq '.updates'`,
	Args: cobra.ExactArgs(1),
	RunE: runDisruptionsShow,
}

var disruptionsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a disruption",
	Example: `  larm disruptions create --title "API outage" --impact critical
  larm disruptions create --title "Maintenance" --type maintenance --message "Upgrading database"`,
	RunE: runDisruptionsCreate,
}

var disruptionsUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a disruption",
	Example: `  larm disruptions update <id> --message "Fix deployed, monitoring"
  larm disruptions update <id> --status resolved`,
	Args: cobra.ExactArgs(1),
	RunE: runDisruptionsUpdate,
}

func init() {
	disruptionsCreateCmd.Flags().String("title", "", "Disruption title")
	disruptionsCreateCmd.Flags().String("type", "disruption", "Type: disruption or maintenance")
	disruptionsCreateCmd.Flags().String("impact", "major", "Impact: minor, major, or critical")
	disruptionsCreateCmd.Flags().String("message", "", "Initial timeline entry")
	disruptionsCreateCmd.Flags().String("status-page-id", "", "Publish to this status page")
	_ = disruptionsCreateCmd.MarkFlagRequired("title")

	disruptionsUpdateCmd.Flags().String("title", "", "New title")
	disruptionsUpdateCmd.Flags().String("impact", "", "New impact")
	disruptionsUpdateCmd.Flags().String("status", "", "New status (investigating, identified, monitoring, resolved)")
	disruptionsUpdateCmd.Flags().String("message", "", "Add a timeline entry")
	disruptionsUpdateCmd.Flags().String("status-page-id", "", "Publish to this status page")

	disruptionsCmd.AddCommand(disruptionsListCmd)
	disruptionsCmd.AddCommand(disruptionsShowCmd)
	disruptionsCmd.AddCommand(disruptionsCreateCmd)
	disruptionsCmd.AddCommand(disruptionsUpdateCmd)
	rootCmd.AddCommand(disruptionsCmd)
}

func runDisruptionsList(cmd *cobra.Command, _ []string) error {
	c, err := newTypedClient(cmd)
	if err != nil {
		return err
	}

	resp, err := c.ListDisruptionsWithResponse(cmd.Context())
	if err != nil {
		return err
	}

	return handleAndPrintWithDefaults(cmd, resp.StatusCode(), resp.Body,
		"id,title,status,impact,started_at",
		output.PrintOpts{ColorHints: map[string]output.ColorFunc{
			"status": output.StatusColor,
			"impact": output.StatusColor,
		}})
}

func runDisruptionsShow(cmd *cobra.Command, args []string) error {
	c, err := newTypedClient(cmd)
	if err != nil {
		return err
	}

	id, err := parseUUID(args[0])
	if err != nil {
		return err
	}

	resp, err := c.GetDisruptionWithResponse(cmd.Context(), id)
	if err != nil {
		return err
	}

	return handleAndPrintWithDefaults(cmd, resp.StatusCode(), resp.Body, "",
		output.PrintOpts{ViewFunc: viewDisruption})
}

func runDisruptionsCreate(cmd *cobra.Command, _ []string) error {
	c, err := newTypedClient(cmd)
	if err != nil {
		return err
	}

	title, _ := cmd.Flags().GetString("title")
	dispType, _ := cmd.Flags().GetString("type")
	impact, _ := cmd.Flags().GetString("impact")
	message, _ := cmd.Flags().GetString("message")
	statusPageID, _ := cmd.Flags().GetString("status-page-id")

	dt := client.DisruptionType(dispType)
	imp := client.DisruptionImpact(impact)
	body := client.CreateDisruptionJSONRequestBody{
		Title:  &title,
		Type:   &dt,
		Impact: &imp,
	}

	if message != "" {
		body.Message = &message
	}
	if statusPageID != "" {
		spID, err := parseUUID(statusPageID)
		if err != nil {
			return err
		}
		body.StatusPageId = &spID
	}

	resp, err := c.CreateDisruptionWithResponse(cmd.Context(), body)
	if err != nil {
		return err
	}

	return handleAndPrintWithDefaults(cmd, resp.StatusCode(), resp.Body, "",
		output.PrintOpts{ViewFunc: viewDisruption})
}

func runDisruptionsUpdate(cmd *cobra.Command, args []string) error {
	c, err := newTypedClient(cmd)
	if err != nil {
		return err
	}

	id, err := parseUUID(args[0])
	if err != nil {
		return err
	}

	body := client.UpdateDisruptionJSONRequestBody{}

	if cmd.Flags().Changed("title") {
		title, _ := cmd.Flags().GetString("title")
		body.Title = &title
	}
	if cmd.Flags().Changed("impact") {
		impact, _ := cmd.Flags().GetString("impact")
		imp := client.DisruptionImpact(impact)
		body.Impact = &imp
	}
	if cmd.Flags().Changed("status") {
		status, _ := cmd.Flags().GetString("status")
		st := client.DisruptionStatus(status)
		body.Status = &st
	}
	if cmd.Flags().Changed("message") {
		message, _ := cmd.Flags().GetString("message")
		body.Message = &message
	}
	if cmd.Flags().Changed("status-page-id") {
		spIDRaw, _ := cmd.Flags().GetString("status-page-id")
		spID, err := parseUUID(spIDRaw)
		if err != nil {
			return err
		}
		body.StatusPageId = &spID
	}

	resp, err := c.UpdateDisruptionWithResponse(cmd.Context(), id, body)
	if err != nil {
		return err
	}

	return handleAndPrintWithDefaults(cmd, resp.StatusCode(), resp.Body, "",
		output.PrintOpts{ViewFunc: viewDisruption})
}

func viewDisruption(w io.Writer, data json.RawMessage) error {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	rows := []viewRow{
		field("id", "id"),
		field("title", "title"),
		field("type", "type"),
		colorField("status", "status", output.StatusColor),
		colorField("impact", "impact", output.StatusColor),
		field("started_at", "started_at"),
		field("resolved_at", "resolved_at"),
		field("inserted_at", "inserted_at"),
	}

	if comps := joinStrings(m["affected_components"], ""); comps != "" {
		rows = append(rows, staticField("components", comps))
	}

	if updates, ok := m["updates"].([]any); ok && len(updates) > 0 {
		rows = append(rows, separator())
		for _, u := range updates {
			if upd, ok := u.(map[string]any); ok {
				status := fmt.Sprintf("%v", upd["status"])
				body, _ := upd["body"].(string)
				if body == "" {
					body = "(no message)"
				}
				ts := output.HumanizeTimestamp(fmt.Sprintf("%v", upd["posted_at"]))
				rows = append(rows, staticField(
					fmt.Sprintf("  [%s]", status),
					fmt.Sprintf("%s (%s)", body, ts),
				))
			}
		}
	}

	return writeView(w, m, rows)
}
