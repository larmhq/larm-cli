package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/larmhq/larm-go/client"

	"github.com/larmhq/larm-cli/internal/output"
)

var alertsCmd = &cobra.Command{
	Use:   "alerts",
	Short: "Manage alert channels",
}

var alertsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List alert channels",
	Example: `  larm alerts list
  larm alerts list --output json --jq '.[].name'`,
	RunE: runAlertsList,
}

var alertsShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show an alert channel",
	Args:  cobra.ExactArgs(1),
	RunE:  runAlertsShow,
}

var alertsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an alert channel",
	Example: `  larm alerts create --name "Slack" --type slack --config '{"webhook_url":"https://hooks.slack.com/..."}'
  larm alerts create --name "Email" --type email --default`,
	RunE: runAlertsCreate,
}

var alertsUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update an alert channel",
	Args:  cobra.ExactArgs(1),
	RunE:  runAlertsUpdate,
}

var alertsDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete an alert channel",
	Args:  cobra.ExactArgs(1),
	RunE:  runAlertsDelete,
}

func init() {
	alertsCreateCmd.Flags().String("name", "", "Channel name")
	alertsCreateCmd.Flags().String("type", "", "Channel type (webhook, slack, discord, email, etc.)")
	alertsCreateCmd.Flags().String("config", "", "Channel config as JSON")
	alertsCreateCmd.Flags().Bool("default", false, "Set as default for new monitors")
	_ = alertsCreateCmd.MarkFlagRequired("name")
	_ = alertsCreateCmd.MarkFlagRequired("type")

	alertsUpdateCmd.Flags().String("name", "", "New channel name")
	alertsUpdateCmd.Flags().String("enabled", "", "Enable or disable (true, false)")

	alertsCmd.AddCommand(alertsListCmd)
	alertsCmd.AddCommand(alertsShowCmd)
	alertsCmd.AddCommand(alertsCreateCmd)
	alertsCmd.AddCommand(alertsUpdateCmd)
	alertsCmd.AddCommand(alertsDeleteCmd)
	rootCmd.AddCommand(alertsCmd)
}

func runAlertsList(cmd *cobra.Command, _ []string) error {
	c, err := newTypedClient(cmd)
	if err != nil {
		return err
	}

	resp, err := c.ListAlertChannelsWithResponse(cmd.Context())
	if err != nil {
		return err
	}

	return handleAndPrintWithDefaults(cmd, resp.StatusCode(), resp.Body,
		"id,name,type,enabled",
		output.PrintOpts{ColorHints: map[string]output.ColorFunc{
			"enabled": output.BoolColor,
		}})
}

func runAlertsShow(cmd *cobra.Command, args []string) error {
	c, err := newTypedClient(cmd)
	if err != nil {
		return err
	}

	id, err := parseUUID(args[0])
	if err != nil {
		return err
	}

	resp, err := c.GetAlertChannelWithResponse(cmd.Context(), id)
	if err != nil {
		return err
	}

	return handleAndPrintWithDefaults(cmd, resp.StatusCode(), resp.Body, "",
		output.PrintOpts{ViewFunc: viewAlertChannel})
}

func viewAlertChannel(w io.Writer, data json.RawMessage) error {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	return writeView(w, m, []viewRow{
		field("id", "id"),
		field("name", "name"),
		field("type", "type"),
		colorField("enabled", "enabled", output.BoolColor),
		colorField("default_for_new_monitors", "default_for_new_monitors", output.BoolColor),
		field("broken_at", "broken_at"),
		field("broken_reason", "broken_reason"),
		field("inserted_at", "inserted_at"),
		field("updated_at", "updated_at"),
	})
}

func runAlertsCreate(cmd *cobra.Command, _ []string) error {
	c, err := newTypedClient(cmd)
	if err != nil {
		return err
	}

	name, _ := cmd.Flags().GetString("name")
	channelType, _ := cmd.Flags().GetString("type")
	configJSON, _ := cmd.Flags().GetString("config")
	isDefault, _ := cmd.Flags().GetBool("default")

	ct := client.AlertChannelType(channelType)
	body := client.CreateAlertChannelJSONRequestBody{
		Name:                  &name,
		Type:                  &ct,
		DefaultForNewMonitors: &isDefault,
	}

	if configJSON != "" {
		var cfg map[string]interface{}
		if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
			return fmt.Errorf("invalid --config JSON: %w", err)
		}
		body.Config = &cfg
	}

	resp, err := c.CreateAlertChannelWithResponse(cmd.Context(), body)
	if err != nil {
		return err
	}

	return handleAndPrintWithDefaults(cmd, resp.StatusCode(), resp.Body, "",
		output.PrintOpts{ViewFunc: viewAlertChannel})
}

func runAlertsUpdate(cmd *cobra.Command, args []string) error {
	c, err := newTypedClient(cmd)
	if err != nil {
		return err
	}

	id, err := parseUUID(args[0])
	if err != nil {
		return err
	}

	body := client.UpdateAlertChannelJSONRequestBody{}

	if cmd.Flags().Changed("name") {
		name, _ := cmd.Flags().GetString("name")
		body.Name = &name
	}
	if cmd.Flags().Changed("enabled") {
		en, _ := cmd.Flags().GetString("enabled")
		enabled := en == "true"
		body.Enabled = &enabled
	}

	resp, err := c.UpdateAlertChannelWithResponse(cmd.Context(), id, body)
	if err != nil {
		return err
	}

	return handleAndPrintWithDefaults(cmd, resp.StatusCode(), resp.Body, "",
		output.PrintOpts{ViewFunc: viewAlertChannel})
}

func runAlertsDelete(cmd *cobra.Command, args []string) error {
	if err := confirmAction(cmd, fmt.Sprintf("Delete alert channel %s?", args[0])); err != nil {
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

	resp, err := c.DeleteAlertChannelWithResponse(cmd.Context(), id)
	if err != nil {
		return err
	}

	return handleDelete(cmd, resp.StatusCode(), resp.Body)
}
