package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/larmhq/larm-go/client"

	"github.com/larmhq/larm-cli/internal/output"
)

var statusPagesCmd = &cobra.Command{
	Use:   "status-pages",
	Short: "Manage status pages",
}

var statusPagesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List status pages",
	Example: `  larm status-pages list
  larm status-pages list --fields name,slug,url`,
	RunE: runStatusPagesList,
}

var statusPagesShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a status page",
	Args:  cobra.ExactArgs(1),
	RunE:  runStatusPagesShow,
}

var statusPagesCreateCmd = &cobra.Command{
	Use:     "create",
	Short:   "Create a status page",
	Example: `  larm status-pages create --name "Acme Status" --slug acme-status`,
	RunE:    runStatusPagesCreate,
}

var statusPagesUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a status page",
	Args:  cobra.ExactArgs(1),
	RunE:  runStatusPagesUpdate,
}

var statusPagesDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a status page",
	Args:  cobra.ExactArgs(1),
	RunE:  runStatusPagesDelete,
}

func init() {
	statusPagesCreateCmd.Flags().String("name", "", "Status page name")
	statusPagesCreateCmd.Flags().String("slug", "", "URL slug (3-63 chars, lowercase alphanumeric and hyphens)")
	_ = statusPagesCreateCmd.MarkFlagRequired("name")
	_ = statusPagesCreateCmd.MarkFlagRequired("slug")

	statusPagesUpdateCmd.Flags().String("name", "", "New name")
	statusPagesUpdateCmd.Flags().String("enabled", "", "Enable or disable (true, false)")
	statusPagesUpdateCmd.Flags().String("description", "", "Page description")

	statusPagesCmd.AddCommand(statusPagesListCmd)
	statusPagesCmd.AddCommand(statusPagesShowCmd)
	statusPagesCmd.AddCommand(statusPagesCreateCmd)
	statusPagesCmd.AddCommand(statusPagesUpdateCmd)
	statusPagesCmd.AddCommand(statusPagesDeleteCmd)
	rootCmd.AddCommand(statusPagesCmd)
}

func runStatusPagesList(cmd *cobra.Command, _ []string) error {
	c, err := newTypedClient(cmd)
	if err != nil {
		return err
	}

	resp, err := c.ListStatusPagesWithResponse(cmd.Context())
	if err != nil {
		return err
	}

	return handleAndPrintWithDefaults(cmd, resp.StatusCode(), resp.Body,
		"id,name,slug,enabled,url",
		output.PrintOpts{ColorHints: map[string]output.ColorFunc{
			"enabled": output.BoolColor,
		}})
}

func runStatusPagesShow(cmd *cobra.Command, args []string) error {
	c, err := newTypedClient(cmd)
	if err != nil {
		return err
	}

	id, err := parseUUID(args[0])
	if err != nil {
		return err
	}

	resp, err := c.GetStatusPageWithResponse(cmd.Context(), id)
	if err != nil {
		return err
	}

	return handleAndPrintWithDefaults(cmd, resp.StatusCode(), resp.Body, "",
		output.PrintOpts{ViewFunc: viewStatusPage})
}

func runStatusPagesCreate(cmd *cobra.Command, _ []string) error {
	c, err := newTypedClient(cmd)
	if err != nil {
		return err
	}

	name, _ := cmd.Flags().GetString("name")
	slug, _ := cmd.Flags().GetString("slug")

	body := client.CreateStatusPageJSONRequestBody{
		Name: &name,
		Slug: &slug,
	}

	resp, err := c.CreateStatusPageWithResponse(cmd.Context(), body)
	if err != nil {
		return err
	}

	return handleAndPrintWithDefaults(cmd, resp.StatusCode(), resp.Body, "",
		output.PrintOpts{ViewFunc: viewStatusPage})
}

func runStatusPagesUpdate(cmd *cobra.Command, args []string) error {
	c, err := newTypedClient(cmd)
	if err != nil {
		return err
	}

	id, err := parseUUID(args[0])
	if err != nil {
		return err
	}

	body := client.UpdateStatusPageJSONRequestBody{}

	if cmd.Flags().Changed("name") {
		name, _ := cmd.Flags().GetString("name")
		body.Name = &name
	}
	if cmd.Flags().Changed("enabled") {
		en, _ := cmd.Flags().GetString("enabled")
		enabled := en == "true"
		body.Enabled = &enabled
	}
	if cmd.Flags().Changed("description") {
		desc, _ := cmd.Flags().GetString("description")
		body.Description = &desc
	}

	resp, err := c.UpdateStatusPageWithResponse(cmd.Context(), id, body)
	if err != nil {
		return err
	}

	return handleAndPrintWithDefaults(cmd, resp.StatusCode(), resp.Body, "",
		output.PrintOpts{ViewFunc: viewStatusPage})
}

func runStatusPagesDelete(cmd *cobra.Command, args []string) error {
	if err := confirmAction(cmd, fmt.Sprintf("Delete status page %s?", args[0])); err != nil {
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

	resp, err := c.DeleteStatusPageWithResponse(cmd.Context(), id)
	if err != nil {
		return err
	}

	return handleDelete(cmd, resp.StatusCode(), resp.Body)
}

func viewStatusPage(w io.Writer, data json.RawMessage) error {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	rows := []viewRow{
		field("id", "id"),
		field("name", "name"),
		field("slug", "slug"),
		field("url", "url"),
		colorField("enabled", "enabled", output.BoolColor),
		colorField("subscribers_enabled", "subscribers_enabled", output.BoolColor),
		field("theme", "theme"),
		field("primary_color", "primary_color"),
		field("description", "description"),
		staticField("components", joinStrings(m["components"], "(none)")),
		field("custom_domain", "custom_domain"),
		field("domain_status", "domain_status"),
		field("inserted_at", "inserted_at"),
		field("updated_at", "updated_at"),
	}

	return writeView(w, m, rows)
}
