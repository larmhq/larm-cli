package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/larmhq/larm-cli/internal/api"
	"github.com/larmhq/larm-cli/internal/auth"
	"github.com/larmhq/larm-cli/internal/client"
	"github.com/larmhq/larm-cli/internal/output"
)

// ErrDryRun is returned when --dry-run is set. Not a real error.
var ErrDryRun = fmt.Errorf("dry run")

// resolveAuth extracts flag values and resolves auth.
func resolveAuth(cmd *cobra.Command) (string, error) {
	flagKey, _ := cmd.Flags().GetString("api-key")
	apiURL, _ := cmd.Flags().GetString("api-url")
	return auth.Resolve(flagKey, apiURL)
}

// newTypedClient creates a typed oapi-codegen client from the command flags.
// If --dry-run is set, the client prints the request and returns ErrDryRun.
func newTypedClient(cmd *cobra.Command) (*client.ClientWithResponses, error) {
	key, err := resolveAuth(cmd)
	if err != nil {
		return nil, err
	}

	baseURL, _ := cmd.Flags().GetString("api-url")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	var httpClient *http.Client
	if dryRun {
		httpClient = &http.Client{
			Timeout:   30 * time.Second,
			Transport: &dryRunTransport{},
		}
	} else {
		httpClient = &http.Client{
			Timeout:   30 * time.Second,
			Transport: &api.RetryTransport{},
		}
	}

	return client.NewClientWithResponses(
		baseURL+"/api/v1",
		client.WithHTTPClient(httpClient),
		client.WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
			req.Header.Set("Authorization", "Bearer "+key)
			req.Header.Set("User-Agent", "larm-cli/"+version)
			return nil
		}),
	)
}

type dryRunTransport struct{}

func (t *dryRunTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	fmt.Fprintf(os.Stderr, "%s %s\n", req.Method, req.URL)

	if req.Body != nil {
		body, err := io.ReadAll(req.Body)
		req.Body.Close()
		if err == nil && len(body) > 0 {
			var pretty json.RawMessage
			if json.Unmarshal(body, &pretty) == nil {
				formatted, _ := json.MarshalIndent(pretty, "", "  ")
				fmt.Fprintln(os.Stderr, string(formatted))
			} else {
				fmt.Fprintln(os.Stderr, string(body))
			}
		}
	}

	return nil, ErrDryRun
}

// getOutputFlags reads the shared output flags from a command.
// Auto-detects format: if stdout is not a TTY and user didn't explicitly set --output,
// defaults to JSON for piped/scripted usage.
func getOutputFlags(cmd *cobra.Command) (format, jqExpr, fields string) {
	format, _ = cmd.Flags().GetString("output")
	jqExpr, _ = cmd.Flags().GetString("jq")
	fields, _ = cmd.Flags().GetString("fields")

	if !cmd.Flags().Changed("output") {
		if output.IsTTY() {
			format = "table"
		} else {
			format = "json"
		}
	}

	return
}

// printOutput wraps output.Print, respecting the --quiet flag.
func printOutput(cmd *cobra.Command, format, jqExpr, fields string, data json.RawMessage, opts ...output.PrintOpts) error {
	quiet, _ := cmd.Flags().GetBool("quiet")
	if quiet {
		return nil
	}
	return output.Print(os.Stdout, format, jqExpr, fields, data, opts...)
}

// handleAndPrint checks the response status, unwraps data, and prints.
func handleAndPrint(cmd *cobra.Command, statusCode int, body []byte) error {
	return handleAndPrintWithDefaults(cmd, statusCode, body, "", output.PrintOpts{})
}

// confirmAction prompts for confirmation in TTY mode. Returns nil if confirmed,
// error if denied or --yes was not passed in non-TTY mode.
func confirmAction(cmd *cobra.Command, message string) error {
	yes, _ := cmd.Flags().GetBool("yes")
	if yes {
		return nil
	}

	if !output.IsTTY() {
		return fmt.Errorf("use --yes to confirm in non-interactive mode")
	}

	fmt.Fprintf(os.Stderr, "%s [y/N] ", message)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if answer == "y" || answer == "yes" {
			return nil
		}
	}
	return fmt.Errorf("cancelled")
}

// handleDelete handles 204 No Content responses from delete endpoints.
func handleDelete(cmd *cobra.Command, statusCode int, body []byte) error {
	if statusCode == 204 {
		quiet, _ := cmd.Flags().GetBool("quiet")
		if !quiet {
			fmt.Fprintln(os.Stdout, "Deleted")
		}
		return nil
	}
	return handleAndPrint(cmd, statusCode, body)
}

// handleAndPrintWithDefaults is like handleAndPrint but applies default table fields.
func handleAndPrintWithDefaults(cmd *cobra.Command, statusCode int, body []byte, defaultFields string, opts ...output.PrintOpts) error {
	if err := output.HandleResponse(statusCode, body); err != nil {
		return err
	}

	data, err := output.UnwrapData(body)
	if err != nil {
		return err
	}

	format, jqExpr, fields := getOutputFlags(cmd)
	if fields == "" && format == "table" && defaultFields != "" {
		fields = defaultFields
	}
	return printOutput(cmd, format, jqExpr, fields, data, opts...)
}
