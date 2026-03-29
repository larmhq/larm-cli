package cmd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/larmhq/larm-cli/internal/api"
	"github.com/larmhq/larm-cli/internal/auth"
	"github.com/larmhq/larm-cli/internal/config"
)

var authHTTPClient = &http.Client{Timeout: 30 * time.Second}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with the Larm API",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in via browser (device flow OAuth)",
	Long:  "Opens a browser for authentication. Use --with-token to paste an API key instead.",
	RunE:  runAuthLogin,
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	RunE:  runAuthStatus,
}

var authTokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Print the current API key or access token",
	RunE:  runAuthToken,
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear stored credentials",
	RunE:  runAuthLogout,
}

func init() {
	authLoginCmd.Flags().Bool("with-token", false, "Read API key from stdin instead of using browser flow")

	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authTokenCmd)
	rootCmd.AddCommand(authCmd)
}

func runAuthLogin(cmd *cobra.Command, _ []string) error {
	withToken, _ := cmd.Flags().GetBool("with-token")
	if withToken {
		return runAuthLoginWithToken(cmd)
	}
	return runAuthLoginDeviceFlow(cmd)
}

func runAuthLoginWithToken(cmd *cobra.Command) error {
	var key string

	if term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Fprint(os.Stderr, "Paste your API key: ")
		raw, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return fmt.Errorf("reading API key: %w", err)
		}
		key = strings.TrimSpace(string(raw))
	} else {
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			key = strings.TrimSpace(scanner.Text())
		}
	}

	if key == "" {
		return fmt.Errorf("no API key provided")
	}

	baseURL, _ := cmd.Flags().GetString("api-url")
	client := api.NewClient(baseURL, key)
	identity, err := client.GetIdentity(cmd.Context())
	if err != nil {
		return fmt.Errorf("key verification failed: %w", err)
	}

	if err := config.Set(config.KeyAPIKey, key); err != nil {
		return fmt.Errorf("saving API key: %w", err)
	}
	// Clear OAuth tokens when switching to API key auth
	_ = config.Set(config.KeyAccessToken, "")
	_ = config.Set(config.KeyRefreshToken, "")
	_ = config.Set(config.KeyExpiresAt, "")

	fmt.Fprintf(os.Stderr, "Logged in to %s\n", identity.Data.Organization.Name)
	fmt.Fprintf(os.Stderr, "Config saved to %s\n", config.Path())
	return nil
}

type deviceCodeResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func runAuthLoginDeviceFlow(cmd *cobra.Command) error {
	baseURL, _ := cmd.Flags().GetString("api-url")

	// Check if already logged in
	if existing, _ := resolveAuth(cmd); existing != "" {
		if term.IsTerminal(int(os.Stdin.Fd())) {
			fmt.Fprint(os.Stderr, "Already logged in. Log in again? [y/N] ")
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
				if answer != "y" && answer != "yes" {
					return nil
				}
			}
		}
	}

	ctx := cmd.Context()

	codeBody, _ := json.Marshal(map[string]string{"client_id": auth.ClientID})
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/oauth/device/code", bytes.NewReader(codeBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := authHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to request device code: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to request device code: %s", string(body))
	}

	var dc deviceCodeResponse
	if err := json.Unmarshal(body, &dc); err != nil {
		return fmt.Errorf("failed to parse device code response: %w", err)
	}

	// Print instructions
	fmt.Fprintf(os.Stderr, "\nOpen this URL in your browser (the code is pre-filled):\n")
	fmt.Fprintf(os.Stderr, "  %s\n\n", dc.VerificationURIComplete)
	fmt.Fprintf(os.Stderr, "If the code is not pre-filled, enter: %s\n\n", dc.UserCode)

	// Best-effort browser open
	openBrowser(dc.VerificationURIComplete)

	fmt.Fprintf(os.Stderr, "Waiting for you to approve in the browser... (press Ctrl+C to cancel)\n")

	// Poll for token
	interval := time.Duration(dc.Interval) * time.Second
	deadline := time.Now().Add(time.Duration(dc.ExpiresIn) * time.Second)

	for time.Now().Before(deadline) {
		select {
		case <-time.After(interval):
		case <-ctx.Done():
			return nil
		}

		result, pollErr := pollForToken(ctx, baseURL, dc.DeviceCode)
		if pollErr != nil {
			return pollErr
		}

		switch result.status {
		case "ok":
			expiresAt := time.Now().Add(time.Duration(result.tokens.ExpiresIn) * time.Second)

			// Clear API key when switching to OAuth auth
			_ = config.Set(config.KeyAPIKey, "")
			_ = config.Set(config.KeyAccessToken, result.tokens.AccessToken)
			_ = config.Set(config.KeyRefreshToken, result.tokens.RefreshToken)
			_ = config.Set(config.KeyExpiresAt, expiresAt.Format(time.RFC3339))

			client := api.NewClient(baseURL, result.tokens.AccessToken)
			identity, err := client.GetIdentity(ctx)
			if err == nil {
				_ = config.Set(config.KeyOrgName, identity.Data.Organization.Name)
				fmt.Fprintf(os.Stderr, "\nLogged in to %s\n", identity.Data.Organization.Name)
			} else {
				fmt.Fprintf(os.Stderr, "\nLogged in\n")
			}
			fmt.Fprintf(os.Stderr, "Config saved to %s\n", config.Path())
			return nil

		case "authorization_pending":
			continue

		case "slow_down":
			interval += 5 * time.Second
			continue

		case "expired_token":
			return fmt.Errorf("login timed out. Run `larm auth login` to try again")

		case "access_denied":
			return fmt.Errorf("authorization denied")

		default:
			return fmt.Errorf("unexpected error: %s", result.status)
		}
	}

	return fmt.Errorf("login timed out. Run `larm auth login` to try again")
}

type pollResult struct {
	status string
	tokens *auth.TokenResponse
}

func pollForToken(ctx context.Context, baseURL, deviceCode string) (*pollResult, error) {
	payload, _ := auth.MarshalTokenRequest("urn:ietf:params:oauth:grant-type:device_code", "", deviceCode)

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/oauth/device/token", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := authHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("polling failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode == 200 {
		var tokens auth.TokenResponse
		if err := json.Unmarshal(body, &tokens); err != nil {
			return nil, fmt.Errorf("failed to parse token response: %w", err)
		}
		return &pollResult{status: "ok", tokens: &tokens}, nil
	}

	var errResp errorResponse
	if err := json.Unmarshal(body, &errResp); err != nil {
		return nil, fmt.Errorf("failed to parse error response: %s", string(body))
	}

	return &pollResult{status: errResp.Error}, nil
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return
	}
	_ = cmd.Start()
}

func runAuthStatus(cmd *cobra.Command, _ []string) error {
	key, err := resolveAuth(cmd)
	if err != nil {
		return err
	}

	baseURL, _ := cmd.Flags().GetString("api-url")
	client := api.NewClient(baseURL, key)
	identity, err := client.GetIdentity(cmd.Context())
	if err != nil {
		return err
	}

	method := "API key"
	if config.Get(config.KeyAccessToken) != "" {
		method = "OAuth token"
	}
	fmt.Fprintf(os.Stdout, "Authenticated to %s (via %s)\n", identity.Data.Organization.Name, method)
	return nil
}

func runAuthToken(cmd *cobra.Command, _ []string) error {
	key, err := resolveAuth(cmd)
	if err != nil {
		return err
	}

	fmt.Fprintln(os.Stdout, key)
	return nil
}

func runAuthLogout(_ *cobra.Command, _ []string) error {
	for _, key := range []string{
		config.KeyAPIKey,
		config.KeyAccessToken,
		config.KeyRefreshToken,
		config.KeyExpiresAt,
		config.KeyOrgName,
	} {
		_ = config.Set(key, "")
	}
	fmt.Fprintln(os.Stderr, "Logged out")
	return nil
}
