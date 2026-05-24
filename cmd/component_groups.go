package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/larmhq/larm-go/client"

	"github.com/larmhq/larm-cli/internal/output"
)

var componentGroupsCmd = &cobra.Command{
	Use:   "component-groups",
	Short: "Manage status page component groups",
}

var componentGroupsShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a component group",
	Args:  cobra.ExactArgs(1),
	RunE:  runComponentGroupsShow,
}

var componentGroupsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a component group",
	RunE:  runComponentGroupsCreate,
}

var componentGroupsUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a component group",
	Args:  cobra.ExactArgs(1),
	RunE:  runComponentGroupsUpdate,
}

var componentGroupsDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a component group",
	Long:  "Delete a component group. Linked components are moved to the top level (component_group_id becomes null) — they are not deleted.",
	Args:  cobra.ExactArgs(1),
	RunE:  runComponentGroupsDelete,
}

func init() {
	for _, cmd := range []*cobra.Command{componentGroupsShowCmd, componentGroupsCreateCmd, componentGroupsUpdateCmd, componentGroupsDeleteCmd} {
		cmd.Flags().String("status-page-id", "", "Status page ID (required)")
		_ = cmd.MarkFlagRequired("status-page-id")
	}

	componentGroupsCreateCmd.Flags().String("name", "", "Group name")
	componentGroupsCreateCmd.Flags().Int("position", 0, "Display position")
	_ = componentGroupsCreateCmd.MarkFlagRequired("name")

	componentGroupsUpdateCmd.Flags().String("name", "", "New group name")
	componentGroupsUpdateCmd.Flags().Int("position", 0, "New position")

	componentGroupsCmd.AddCommand(componentGroupsShowCmd)
	componentGroupsCmd.AddCommand(componentGroupsCreateCmd)
	componentGroupsCmd.AddCommand(componentGroupsUpdateCmd)
	componentGroupsCmd.AddCommand(componentGroupsDeleteCmd)
	rootCmd.AddCommand(componentGroupsCmd)
}

func runComponentGroupsShow(cmd *cobra.Command, args []string) error {
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

	resp, err := c.GetComponentGroupWithResponse(cmd.Context(), spID, id)
	if err != nil {
		return err
	}

	return handleAndPrintWithDefaults(cmd, resp.StatusCode(), resp.Body, "",
		output.PrintOpts{ViewFunc: viewComponentGroup})
}

func runComponentGroupsCreate(cmd *cobra.Command, _ []string) error {
	c, err := newTypedClient(cmd)
	if err != nil {
		return err
	}

	spID, err := getStatusPageID(cmd)
	if err != nil {
		return err
	}

	name, _ := cmd.Flags().GetString("name")
	body := client.CreateComponentGroupJSONRequestBody{Name: &name}

	if cmd.Flags().Changed("position") {
		pos, _ := cmd.Flags().GetInt("position")
		body.Position = &pos
	}

	resp, err := c.CreateComponentGroupWithResponse(cmd.Context(), spID, body)
	if err != nil {
		return err
	}

	return handleAndPrintWithDefaults(cmd, resp.StatusCode(), resp.Body, "",
		output.PrintOpts{ViewFunc: viewComponentGroup})
}

func runComponentGroupsUpdate(cmd *cobra.Command, args []string) error {
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

	body := client.UpdateComponentGroupJSONRequestBody{}

	if cmd.Flags().Changed("name") {
		name, _ := cmd.Flags().GetString("name")
		body.Name = &name
	}
	if cmd.Flags().Changed("position") {
		pos, _ := cmd.Flags().GetInt("position")
		body.Position = &pos
	}

	resp, err := c.UpdateComponentGroupWithResponse(cmd.Context(), spID, id, body)
	if err != nil {
		return err
	}

	return handleAndPrintWithDefaults(cmd, resp.StatusCode(), resp.Body, "",
		output.PrintOpts{ViewFunc: viewComponentGroup})
}

func runComponentGroupsDelete(cmd *cobra.Command, args []string) error {
	if err := confirmAction(cmd, fmt.Sprintf("Delete component group %s? Components inside it will be moved to top level.", args[0])); err != nil {
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

	resp, err := c.DeleteComponentGroupWithResponse(cmd.Context(), spID, id)
	if err != nil {
		return err
	}

	return handleDelete(cmd, resp.StatusCode(), resp.Body)
}

func viewComponentGroup(w io.Writer, data json.RawMessage) error {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	return writeView(w, m, []viewRow{
		field("id", "id"),
		field("name", "name"),
		field("position", "position"),
		field("inserted_at", "inserted_at"),
		field("updated_at", "updated_at"),
	})
}
