package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/larmhq/larm-cli/internal/client"
	"github.com/larmhq/larm-cli/internal/output"
)

var webhooksCmd = &cobra.Command{
	Use:   "webhooks",
	Short: "Manage webhook subscriptions",
}

var webhooksListCmd = &cobra.Command{
	Use:   "list",
	Short: "List webhook subscriptions",
	RunE:  runWebhooksList,
}

var webhooksShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a webhook subscription",
	Args:  cobra.ExactArgs(1),
	RunE:  runWebhooksShow,
}

var webhooksCreateCmd = &cobra.Command{
	Use:     "create",
	Short:   "Create a webhook subscription",
	Long:    "Create a webhook subscription. The HMAC signing secret is only returned in the create response.",
	Example: `  larm webhooks create --url https://example.com/hook --events monitor.state_changed`,
	RunE:    runWebhooksCreate,
}

var webhooksUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a webhook subscription",
	Args:  cobra.ExactArgs(1),
	RunE:  runWebhooksUpdate,
}

var webhooksDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a webhook subscription",
	Args:  cobra.ExactArgs(1),
	RunE:  runWebhooksDelete,
}

func init() {
	webhooksCreateCmd.Flags().String("url", "", "Webhook URL (HTTPS only)")
	webhooksCreateCmd.Flags().StringSlice("events", nil, "Events to subscribe to (monitor.state_changed, monitor.created, monitor.updated, monitor.deleted)")
	_ = webhooksCreateCmd.MarkFlagRequired("url")
	_ = webhooksCreateCmd.MarkFlagRequired("events")

	webhooksUpdateCmd.Flags().String("url", "", "New webhook URL")
	webhooksUpdateCmd.Flags().StringSlice("events", nil, "New events list")
	webhooksUpdateCmd.Flags().String("enabled", "", "Enable or disable (true, false)")

	webhooksCmd.AddCommand(webhooksListCmd)
	webhooksCmd.AddCommand(webhooksShowCmd)
	webhooksCmd.AddCommand(webhooksCreateCmd)
	webhooksCmd.AddCommand(webhooksUpdateCmd)
	webhooksCmd.AddCommand(webhooksDeleteCmd)
	rootCmd.AddCommand(webhooksCmd)
}

func runWebhooksList(cmd *cobra.Command, _ []string) error {
	c, err := newTypedClient(cmd)
	if err != nil {
		return err
	}

	resp, err := c.ListWebhooksWithResponse(cmd.Context())
	if err != nil {
		return err
	}

	return handleAndPrintWithDefaults(cmd, resp.StatusCode(), resp.Body,
		"id,url,events,enabled")
}

func runWebhooksShow(cmd *cobra.Command, args []string) error {
	c, err := newTypedClient(cmd)
	if err != nil {
		return err
	}

	id, err := parseUUID(args[0])
	if err != nil {
		return err
	}

	resp, err := c.GetWebhookWithResponse(cmd.Context(), id)
	if err != nil {
		return err
	}

	return handleAndPrintWithDefaults(cmd, resp.StatusCode(), resp.Body, "",
		output.PrintOpts{ViewFunc: viewWebhook})
}

func runWebhooksCreate(cmd *cobra.Command, _ []string) error {
	c, err := newTypedClient(cmd)
	if err != nil {
		return err
	}

	url, _ := cmd.Flags().GetString("url")
	eventsRaw, _ := cmd.Flags().GetStringSlice("events")

	events := make([]client.WebhookEvent, len(eventsRaw))
	for i, e := range eventsRaw {
		events[i] = client.WebhookEvent(e)
	}

	body := client.CreateWebhookJSONRequestBody{
		Url:    &url,
		Events: &events,
	}

	resp, err := c.CreateWebhookWithResponse(cmd.Context(), body)
	if err != nil {
		return err
	}

	return handleAndPrintWithDefaults(cmd, resp.StatusCode(), resp.Body, "",
		output.PrintOpts{ViewFunc: viewWebhook})
}

func runWebhooksUpdate(cmd *cobra.Command, args []string) error {
	c, err := newTypedClient(cmd)
	if err != nil {
		return err
	}

	id, err := parseUUID(args[0])
	if err != nil {
		return err
	}

	body := client.UpdateWebhookJSONRequestBody{}

	if cmd.Flags().Changed("url") {
		url, _ := cmd.Flags().GetString("url")
		body.Url = &url
	}
	if cmd.Flags().Changed("events") {
		eventsRaw, _ := cmd.Flags().GetStringSlice("events")
		events := make([]client.WebhookEvent, len(eventsRaw))
		for i, e := range eventsRaw {
			events[i] = client.WebhookEvent(e)
		}
		body.Events = &events
	}
	if cmd.Flags().Changed("enabled") {
		en, _ := cmd.Flags().GetString("enabled")
		enabled := en == "true"
		body.Enabled = &enabled
	}

	resp, err := c.UpdateWebhookWithResponse(cmd.Context(), id, body)
	if err != nil {
		return err
	}

	return handleAndPrintWithDefaults(cmd, resp.StatusCode(), resp.Body, "",
		output.PrintOpts{ViewFunc: viewWebhook})
}

func runWebhooksDelete(cmd *cobra.Command, args []string) error {
	if err := confirmAction(cmd, fmt.Sprintf("Delete webhook %s?", args[0])); err != nil {
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

	resp, err := c.DeleteWebhookWithResponse(cmd.Context(), id)
	if err != nil {
		return err
	}

	return handleDelete(cmd, resp.StatusCode(), resp.Body)
}

func viewWebhook(w io.Writer, data json.RawMessage) error {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	return writeView(w, m, []viewRow{
		field("id", "id"),
		field("url", "url"),
		staticField("events", joinStrings(m["events"], "(none)")),
		colorField("enabled", "enabled", output.BoolColor),
		field("secret", "secret"),
		field("inserted_at", "inserted_at"),
		field("updated_at", "updated_at"),
	})
}
