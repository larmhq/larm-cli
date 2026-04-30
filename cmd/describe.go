package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var describeCmd = &cobra.Command{
	Use:   "describe [command.subcommand]",
	Short: "Output JSON schema of commands for LLM agent discovery",
	Long: `Outputs a JSON description of all commands, subcommands, and flags.
Useful for LLM agents to discover available functionality.`,
	Example: `  larm describe
  larm describe monitors.list
  larm describe disruptions`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDescribe,
}

func init() {
	rootCmd.AddCommand(describeCmd)
}

type commandSchema struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Usage       string          `json:"usage,omitempty"`
	Subcommands []commandSchema `json:"subcommands,omitempty"`
	Flags       []flagSchema    `json:"flags,omitempty"`
	GlobalFlags []flagSchema    `json:"global_flags,omitempty"`
}

type flagSchema struct {
	Name        string `json:"name"`
	Shorthand   string `json:"shorthand,omitempty"`
	Type        string `json:"type"`
	Default     string `json:"default,omitempty"`
	Description string `json:"description"`
	Required    bool   `json:"required,omitempty"`
}

func runDescribe(_ *cobra.Command, args []string) error {
	if len(args) == 1 {
		return describeCommand(args[0])
	}
	return describeAll()
}

func describeAll() error {
	schema := buildSchema(rootCmd)
	schema.Flags = collectGlobalFlags()
	return printSchema(schema)
}

func describeCommand(path string) error {
	parts := strings.Split(path, ".")
	cmd := rootCmd
	for _, part := range parts {
		found := false
		for _, sub := range cmd.Commands() {
			if sub.Name() == part {
				cmd = sub
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("command not found: %s", path)
		}
	}

	schema := buildSchema(cmd)
	schema.GlobalFlags = collectGlobalFlags()
	return printSchema(schema)
}

func buildSchema(cmd *cobra.Command) commandSchema {
	s := commandSchema{
		Name:        cmd.Name(),
		Description: cmd.Short,
		Usage:       cmd.UseLine(),
	}

	for _, sub := range cmd.Commands() {
		if sub.Hidden || sub.Name() == "help" || sub.Name() == "completion" {
			continue
		}
		s.Subcommands = append(s.Subcommands, buildSchema(sub))
	}

	cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
		if f.Hidden {
			return
		}
		fs := flagSchema{
			Name:        f.Name,
			Type:        f.Value.Type(),
			Description: f.Usage,
		}
		if f.Shorthand != "" {
			fs.Shorthand = f.Shorthand
		}
		if f.DefValue != "" && f.DefValue != "false" && f.DefValue != "0" && f.DefValue != "[]" {
			fs.Default = f.DefValue
		}
		s.Flags = append(s.Flags, fs)
	})

	return s
}

func collectGlobalFlags() []flagSchema {
	var flags []flagSchema
	rootCmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		if f.Hidden || f.Name == "help" {
			return
		}
		fs := flagSchema{
			Name:        f.Name,
			Type:        f.Value.Type(),
			Description: f.Usage,
		}
		if f.Shorthand != "" {
			fs.Shorthand = f.Shorthand
		}
		if f.DefValue != "" && f.DefValue != "false" && f.DefValue != "0" {
			fs.Default = f.DefValue
		}
		flags = append(flags, fs)
	})
	return flags
}

func printSchema(s commandSchema) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(s)
}
