package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/larmhq/larm-cli/internal/api"
	"github.com/larmhq/larm-cli/internal/output"
)

var apiCmd = &cobra.Command{
	Use:   "api <method> <path>",
	Short: "Make a raw API request",
	Long: `Hit any Larm API endpoint directly.

For GET/HEAD, --field adds query parameters.
For POST/PUT/PATCH, --field builds a JSON body.
Use --input - to read body from stdin.`,
	Example: `  larm api GET /monitors
  larm api GET /monitors --field check_type=http
  larm api POST /monitors --field name=Test --field check_type=http
  echo '{"name":"Test"}' | larm api POST /monitors --input -`,
	Args: cobra.ExactArgs(2),
	RunE: runAPI,
}

func init() {
	apiCmd.Flags().StringArray("field", nil, "Key=value pair (query param for GET, JSON body field for POST/PATCH)")
	apiCmd.Flags().String("input", "", "Read body from file (use - for stdin)")
	rootCmd.AddCommand(apiCmd)
}

func runAPI(cmd *cobra.Command, args []string) error {
	method := strings.ToUpper(args[0])
	path := args[1]

	key, err := resolveAuth(cmd)
	if err != nil {
		return err
	}

	baseURL, _ := cmd.Flags().GetString("api-url")
	fields, _ := cmd.Flags().GetStringArray("field")
	inputFile, _ := cmd.Flags().GetString("input")

	url := baseURL + "/api/v1" + path

	var body io.Reader

	if method == "GET" || method == "HEAD" || method == "DELETE" {
		if len(fields) > 0 {
			parsed, err := neturl.Parse(url)
			if err != nil {
				return fmt.Errorf("invalid URL: %w", err)
			}
			q := parsed.Query()
			for _, f := range fields {
				k, v, _ := strings.Cut(f, "=")
				q.Add(k, v)
			}
			parsed.RawQuery = q.Encode()
			url = parsed.String()
		}
	} else {
		// Fields become JSON body
		if inputFile != "" {
			body, err = openInput(inputFile)
			if err != nil {
				return err
			}
		} else if len(fields) > 0 {
			obj := make(map[string]interface{})
			for _, f := range fields {
				k, v, _ := strings.Cut(f, "=")
				obj[k] = v
			}
			b, err := json.Marshal(obj)
			if err != nil {
				return err
			}
			body = bytes.NewReader(b)
		}
	}

	req, err := http.NewRequestWithContext(cmd.Context(), method, url, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("User-Agent", "larm-cli/"+version)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		fmt.Fprintf(os.Stderr, "%s %s\n", method, url)
		return nil
	}

	apiClient := &http.Client{
		Timeout:   30 * time.Second,
		Transport: &api.RetryTransport{},
	}
	resp, err := apiClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		apiErr := output.ParseAPIError(resp.StatusCode, respBody)
		fmt.Fprintln(os.Stderr, apiErr)
		return apiErr
	}

	format, jqExpr, fieldsList := getOutputFlags(cmd)

	// Try to unwrap {"data": ...} envelope, fall back to raw body
	data, err := output.UnwrapData(respBody)
	if err != nil {
		data = respBody
	}

	return printOutput(cmd, format, jqExpr, fieldsList, data)
}

func openInput(path string) (io.Reader, error) {
	if path == "-" {
		return bufio.NewReader(os.Stdin), nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading input file: %w", err)
	}
	return bytes.NewReader(data), nil
}
