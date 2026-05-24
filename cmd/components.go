package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/larmhq/larm-go/client"

	"github.com/larmhq/larm-cli/internal/output"
)

var componentsCmd = &cobra.Command{
	Use:   "components",
	Short: "Manage status page components",
}

var componentsShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a component",
	Args:  cobra.ExactArgs(1),
	RunE:  runComponentsShow,
}

var componentsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a component",
	RunE:  runComponentsCreate,
}

var componentsUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a component",
	Args:  cobra.ExactArgs(1),
	RunE:  runComponentsUpdate,
}

var componentsDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a component",
	Args:  cobra.ExactArgs(1),
	RunE:  runComponentsDelete,
}

func init() {
	// All component commands require --status-page-id
	for _, cmd := range []*cobra.Command{componentsShowCmd, componentsCreateCmd, componentsUpdateCmd, componentsDeleteCmd} {
		cmd.Flags().String("status-page-id", "", "Status page ID (required)")
		_ = cmd.MarkFlagRequired("status-page-id")
	}

	componentsCreateCmd.Flags().String("name", "", "Component name")
	componentsCreateCmd.Flags().String("description", "", "Component description")
	componentsCreateCmd.Flags().Int("position", 0, "Display position")
	componentsCreateCmd.Flags().String("group", "", "Component group ID (puts the component inside this group)")
	_ = componentsCreateCmd.MarkFlagRequired("name")

	componentsUpdateCmd.Flags().String("name", "", "New component name")
	componentsUpdateCmd.Flags().String("description", "", "New description")
	componentsUpdateCmd.Flags().Int("position", 0, "New position")
	componentsUpdateCmd.Flags().String("group", "", "Component group ID (use empty string to move to top level)")

	componentsCmd.AddCommand(componentsShowCmd)
	componentsCmd.AddCommand(componentsCreateCmd)
	componentsCmd.AddCommand(componentsUpdateCmd)
	componentsCmd.AddCommand(componentsDeleteCmd)
	rootCmd.AddCommand(componentsCmd)
}

func getStatusPageID(cmd *cobra.Command) (client.StatusPageId, error) {
	raw, _ := cmd.Flags().GetString("status-page-id")
	return parseUUID(raw)
}

func runComponentsShow(cmd *cobra.Command, args []string) error {
	c, err := newTypedClient(cmd)
	if err != nil {
		return err
	}

	spID, err := getStatusPageID(cmd)
	if err != nil {
		return err
	}

	id, err := parseUUID(args[0])
	if err != nil {
		return err
	}

	resp, err := c.GetComponentWithResponse(cmd.Context(), spID, id)
	if err != nil {
		return err
	}

	return handleAndPrintWithDefaults(cmd, resp.StatusCode(), resp.Body, "",
		output.PrintOpts{ViewFunc: viewComponent})
}

func runComponentsCreate(cmd *cobra.Command, _ []string) error {
	c, err := newTypedClient(cmd)
	if err != nil {
		return err
	}

	spID, err := getStatusPageID(cmd)
	if err != nil {
		return err
	}

	name, _ := cmd.Flags().GetString("name")
	body := client.CreateComponentJSONRequestBody{
		Name: &name,
	}

	if cmd.Flags().Changed("description") {
		desc, _ := cmd.Flags().GetString("description")
		body.Description = &desc
	}
	if cmd.Flags().Changed("position") {
		pos, _ := cmd.Flags().GetInt("position")
		body.Position = &pos
	}
	if cmd.Flags().Changed("group") {
		raw, _ := cmd.Flags().GetString("group")
		groupID, err := parseUUID(raw)
		if err != nil {
			return fmt.Errorf("invalid --group: %w", err)
		}
		body.ComponentGroupId = &groupID
	}

	resp, err := c.CreateComponentWithResponse(cmd.Context(), spID, body)
	if err != nil {
		return err
	}

	return handleAndPrintWithDefaults(cmd, resp.StatusCode(), resp.Body, "",
		output.PrintOpts{ViewFunc: viewComponent})
}

func runComponentsUpdate(cmd *cobra.Command, args []string) error {
	c, err := newTypedClient(cmd)
	if err != nil {
		return err
	}

	spID, err := getStatusPageID(cmd)
	if err != nil {
		return err
	}

	id, err := parseUUID(args[0])
	if err != nil {
		return err
	}

	body := client.UpdateComponentJSONRequestBody{}

	if cmd.Flags().Changed("name") {
		name, _ := cmd.Flags().GetString("name")
		body.Name = &name
	}
	if cmd.Flags().Changed("description") {
		desc, _ := cmd.Flags().GetString("description")
		body.Description = &desc
	}
	if cmd.Flags().Changed("position") {
		pos, _ := cmd.Flags().GetInt("position")
		body.Position = &pos
	}
	if cmd.Flags().Changed("group") {
		raw, _ := cmd.Flags().GetString("group")
		groupID, err := parseUUID(raw)
		if err != nil {
			return fmt.Errorf("invalid --group: %w", err)
		}
		body.ComponentGroupId = &groupID
	}

	resp, err := c.UpdateComponentWithResponse(cmd.Context(), spID, id, body)
	if err != nil {
		return err
	}

	return handleAndPrintWithDefaults(cmd, resp.StatusCode(), resp.Body, "",
		output.PrintOpts{ViewFunc: viewComponent})
}

func runComponentsDelete(cmd *cobra.Command, args []string) error {
	if err := confirmAction(cmd, fmt.Sprintf("Delete component %s?", args[0])); err != nil {
		return err
	}

	c, err := newTypedClient(cmd)
	if err != nil {
		return err
	}

	spID, err := getStatusPageID(cmd)
	if err != nil {
		return err
	}

	id, err := parseUUID(args[0])
	if err != nil {
		return err
	}

	resp, err := c.DeleteComponentWithResponse(cmd.Context(), spID, id)
	if err != nil {
		return err
	}

	return handleDelete(cmd, resp.StatusCode(), resp.Body)
}

func viewComponent(w io.Writer, data json.RawMessage) error {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	return writeView(w, m, []viewRow{
		field("id", "id"),
		field("name", "name"),
		field("description", "description"),
		field("position", "position"),
		staticField("monitors", joinStrings(m["monitors"], "(none)")),
		field("inserted_at", "inserted_at"),
		field("updated_at", "updated_at"),
	})
}
