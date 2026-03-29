package cmd

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/larmhq/larm-cli/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage CLI configuration",
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a config value",
	Args:  cobra.ExactArgs(1),
	RunE:  runConfigGet,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a config value",
	Args:  cobra.ExactArgs(2),
	RunE:  runConfigSet,
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all config values",
	RunE:  runConfigList,
}

func init() {
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configListCmd)
	rootCmd.AddCommand(configCmd)
}

func runConfigGet(_ *cobra.Command, args []string) error {
	value := config.Get(args[0])
	if value == "" {
		return fmt.Errorf("key %q not set", args[0])
	}

	if args[0] == "api_key" {
		value = maskKey(value)
	}
	fmt.Fprintln(os.Stdout, value)
	return nil
}

func runConfigSet(_ *cobra.Command, args []string) error {
	if !config.IsValidKey(args[0]) {
		return fmt.Errorf("unknown config key %q. Valid keys: %s", args[0], strings.Join(config.ValidKeys, ", "))
	}
	if err := config.Set(args[0], args[1]); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Set %s\n", args[0])
	return nil
}

func runConfigList(_ *cobra.Command, _ []string) error {
	all := config.All()
	keys := make([]string, 0, len(all))
	for k := range all {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	for _, k := range keys {
		v := fmt.Sprintf("%v", all[k])
		if k == "api_key" {
			v = maskKey(v)
		}
		fmt.Fprintf(os.Stdout, "%s: %s\n", k, v)
	}
	return nil
}

func maskKey(key string) string {
	if len(key) <= 8 {
		return strings.Repeat("*", len(key))
	}
	return key[:4] + strings.Repeat("*", len(key)-8) + key[len(key)-4:]
}
