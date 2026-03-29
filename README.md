# larm

The official CLI for [Larm](https://larm.dev) -- uptime monitoring for engineering teams.

Manage monitors, alerts, status pages, incidents, and webhooks from your terminal.

## Install

### Homebrew (macOS/Linux)

```
brew install larmhq/tap/larm
```

### Binary download

Download the latest release from [GitHub Releases](https://github.com/larmhq/larm-cli/releases).

### Go install

```
go install github.com/larmhq/larm-cli@latest
```

## Authentication

### Browser login (recommended)

```
larm auth login
```

Opens your browser for device flow OAuth. Enter the code shown in your terminal, pick your organization, and approve.

### API key

```
larm auth login --with-token
```

Paste an API key from [Settings > API Keys](https://app.larm.dev/settings/api-keys).

Or set the `LARM_API_KEY` environment variable:

```
export LARM_API_KEY=larm_api_...
```

### Other auth commands

```
larm auth status       # show current auth method and organization
larm auth token        # print the current token/key to stdout
larm auth logout       # clear stored credentials
```

## Usage

### Monitors

```
larm monitors list
larm monitors list --check-type http
larm monitors show <id>
larm monitors create --name "API" --url https://example.com
larm monitors update <id> --name "New Name"
larm monitors delete <id>
larm monitors state <id>
larm monitors uptime <id> --range 7d
larm monitors response-times <id>
larm monitors cert <id>
```

### Alert channels

```
larm alerts list
larm alerts show <id>
larm alerts create --name "Slack" --type slack --config '{"webhook_url":"..."}'
larm alerts update <id> --enabled false
larm alerts delete <id>
```

### Status pages

```
larm status-pages list
larm status-pages show <id>
larm status-pages create --name "Acme Status" --slug acme-status
larm status-pages delete <id>
larm components list --status-page-id <id>
larm components show <component-id> --status-page-id <id>
```

### Incidents

```
larm incidents list
larm incidents show <id>
larm incidents create --title "API outage" --impact critical --message "Investigating"
larm incidents update <id> --status resolved
```

### Webhooks

```
larm webhooks list
larm webhooks show <id>
larm webhooks create --url https://example.com/hook --events monitor.state_changed
larm webhooks delete <id>
```

### Raw API access

```
larm api GET /monitors
larm api POST /monitors --field name=Test --field check_type=http
echo '{"name":"Test"}' | larm api POST /monitors --input -
```

## Output

By default, table output in a terminal, JSON when piped.

```
larm monitors list                              # table
larm monitors list --output json                # JSON
larm monitors list --output json --jq '.[].name'  # JQ filter
larm monitors list --fields name,check_type     # select columns
```

## Other flags

```
larm monitors list --quiet        # suppress output on success
larm monitors create --dry-run    # print request without sending
larm monitors delete <id> --yes   # skip delete confirmation prompt
```

Delete commands prompt for confirmation in interactive mode. Use `--yes` to skip for scripting.

## Configuration

Config is stored at `~/.config/larm/config.yml`.

```
larm config list
larm config get api_url
larm config set api_url https://app.larm.dev
```

## Development

Prerequisites: [Go 1.26+](https://go.dev/dl/), [mise](https://mise.jdx.dev/) (optional, manages tool versions).

```
git clone https://github.com/larmhq/larm-cli
cd larm-cli
mise install          # or install Go manually
make build            # build to bin/larm
make test             # run tests
make lint             # golangci-lint
make fmt              # format code
make generate         # regenerate typed client from OpenAPI spec
make check-generate   # verify generated code matches committed
make snapshot         # test goreleaser locally
```

See [AGENTS.md](AGENTS.md) for architecture details and how to add new commands.

## For LLM agents

```
larm describe                    # full command schema as JSON
larm describe monitors.list      # single command schema
```

Always use `--output json` for machine-readable output.

## License

MIT
