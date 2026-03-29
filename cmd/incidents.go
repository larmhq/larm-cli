package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/larmhq/larm-cli/internal/client"
	"github.com/larmhq/larm-cli/internal/output"
)

var incidentsCmd = &cobra.Command{
	Use:   "incidents",
	Short: "Manage incidents",
}

var incidentsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List incidents",
	Example: `  larm incidents list
  larm incidents list --output json --jq '[.[] | select(.status != "resolved")]'`,
	RunE: runIncidentsList,
}

var incidentsShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show an incident",
	Example: `  larm incidents show <id>
  larm incidents show <id> --output json --jq '.updates'`,
	Args: cobra.ExactArgs(1),
	RunE: runIncidentsShow,
}

var incidentsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an incident",
	Example: `  larm incidents create --title "API outage" --impact critical
  larm incidents create --title "Maintenance" --type maintenance --message "Upgrading database"`,
	RunE: runIncidentsCreate,
}

var incidentsUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update an incident",
	Example: `  larm incidents update <id> --message "Fix deployed, monitoring"
  larm incidents update <id> --status resolved`,
	Args: cobra.ExactArgs(1),
	RunE: runIncidentsUpdate,
}

func init() {
	incidentsCreateCmd.Flags().String("title", "", "Incident title")
	incidentsCreateCmd.Flags().String("type", "incident", "Type: incident or maintenance")
	incidentsCreateCmd.Flags().String("impact", "major", "Impact: minor, major, or critical")
	incidentsCreateCmd.Flags().String("message", "", "Initial timeline entry")
	incidentsCreateCmd.Flags().String("status-page-id", "", "Publish to this status page")
	_ = incidentsCreateCmd.MarkFlagRequired("title")

	incidentsUpdateCmd.Flags().String("title", "", "New title")
	incidentsUpdateCmd.Flags().String("impact", "", "New impact")
	incidentsUpdateCmd.Flags().String("status", "", "New status (investigating, identified, monitoring, resolved)")
	incidentsUpdateCmd.Flags().String("message", "", "Add a timeline entry")
	incidentsUpdateCmd.Flags().String("status-page-id", "", "Publish to this status page")

	incidentsCmd.AddCommand(incidentsListCmd)
	incidentsCmd.AddCommand(incidentsShowCmd)
	incidentsCmd.AddCommand(incidentsCreateCmd)
	incidentsCmd.AddCommand(incidentsUpdateCmd)
	rootCmd.AddCommand(incidentsCmd)
}

func runIncidentsList(cmd *cobra.Command, _ []string) error {
	c, err := newTypedClient(cmd)
	if err != nil {
		return err
	}

	resp, err := c.ListIncidentsWithResponse(cmd.Context())
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

func runIncidentsShow(cmd *cobra.Command, args []string) error {
	c, err := newTypedClient(cmd)
	if err != nil {
		return err
	}

	id, err := parseUUID(args[0])
	if err != nil {
		return err
	}

	resp, err := c.GetIncidentWithResponse(cmd.Context(), id)
	if err != nil {
		return err
	}

	return handleAndPrintWithDefaults(cmd, resp.StatusCode(), resp.Body, "",
		output.PrintOpts{ViewFunc: viewIncident})
}

func runIncidentsCreate(cmd *cobra.Command, _ []string) error {
	c, err := newTypedClient(cmd)
	if err != nil {
		return err
	}

	title, _ := cmd.Flags().GetString("title")
	incType, _ := cmd.Flags().GetString("type")
	impact, _ := cmd.Flags().GetString("impact")
	message, _ := cmd.Flags().GetString("message")
	statusPageID, _ := cmd.Flags().GetString("status-page-id")

	it := client.IncidentType(incType)
	imp := client.IncidentImpact(impact)
	body := client.CreateIncidentJSONRequestBody{
		Title:  &title,
		Type:   &it,
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

	resp, err := c.CreateIncidentWithResponse(cmd.Context(), body)
	if err != nil {
		return err
	}

	return handleAndPrintWithDefaults(cmd, resp.StatusCode(), resp.Body, "",
		output.PrintOpts{ViewFunc: viewIncident})
}

func runIncidentsUpdate(cmd *cobra.Command, args []string) error {
	c, err := newTypedClient(cmd)
	if err != nil {
		return err
	}

	id, err := parseUUID(args[0])
	if err != nil {
		return err
	}

	body := client.UpdateIncidentJSONRequestBody{}

	if cmd.Flags().Changed("title") {
		title, _ := cmd.Flags().GetString("title")
		body.Title = &title
	}
	if cmd.Flags().Changed("impact") {
		impact, _ := cmd.Flags().GetString("impact")
		imp := client.IncidentImpact(impact)
		body.Impact = &imp
	}
	if cmd.Flags().Changed("status") {
		status, _ := cmd.Flags().GetString("status")
		st := client.IncidentStatus(status)
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

	resp, err := c.UpdateIncidentWithResponse(cmd.Context(), id, body)
	if err != nil {
		return err
	}

	return handleAndPrintWithDefaults(cmd, resp.StatusCode(), resp.Body, "",
		output.PrintOpts{ViewFunc: viewIncident})
}

func viewIncident(w io.Writer, data json.RawMessage) error {
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
