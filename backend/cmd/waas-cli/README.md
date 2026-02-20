# WaaS CLI

Command-line interface for the Webhook-as-a-Service platform. Manage webhook endpoints, send webhooks, view delivery logs, and test integrations from your terminal.

## Installation

### From Source

```bash
cd backend
go build -o waas ./cmd/waas-cli/
# Move to a directory in your PATH
mv waas /usr/local/bin/
```

### Via Go Install

```bash
go install github.com/josedab/waas/cmd/waas-cli@latest
```

### Homebrew (macOS/Linux)

```bash
brew tap waas/tap
brew install waas-cli
```

### Verify Installation

```bash
waas version
```

## Configuration

The CLI reads configuration from (in order of precedence):

1. Command-line flags (`--api-url`, `--api-key`)
2. Environment variables (`WAAS_API_URL`, `WAAS_API_KEY`)
3. Config file `~/.waas.yaml`

### Config File

Create `~/.waas.yaml`:

```yaml
api_url: http://localhost:8080
api_key: your-api-key-here
```

### Global Flags

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--config` | — | `~/.waas.yaml` | Config file path |
| `--api-url` | `WAAS_API_URL` | `http://localhost:8080` | API server URL |
| `--api-key` | `WAAS_API_KEY` | — | API key for authentication |
| `-o, --output` | — | `table` | Output format: `table`, `json`, `yaml`, `csv` |
| `-v, --verbose` | — | `false` | Enable verbose output |

## Commands

### Authentication & Setup

| Command | Description |
|---------|-------------|
| `waas login` | Authenticate with your API key |
| `waas logout` | Remove stored credentials |
| `waas init [directory]` | Initialize a new WaaS project |
| `waas config` | Manage CLI configuration |
| `waas completion [bash\|zsh\|fish\|powershell]` | Generate shell completion scripts |

### Webhook Operations

| Command | Description |
|---------|-------------|
| `waas endpoints` | Manage webhook endpoints (list, create, update, delete) |
| `waas send` | Send a webhook to an endpoint |
| `waas delivery` | Manage webhook deliveries |
| `waas replay <delivery-id>` | Replay a webhook delivery |
| `waas bulk-replay` | Replay multiple failed deliveries |

### Monitoring & Debugging

| Command | Description |
|---------|-------------|
| `waas logs [delivery-id]` | View webhook delivery logs |
| `waas inspect <delivery-id>` | Deep inspect a webhook delivery |
| `waas listen` | Stream webhook events in real-time |
| `waas health` | Check API health status |
| `waas status` | Show account status and usage |

### Testing

| Command | Description |
|---------|-------------|
| `waas test` | Run webhook integration tests |
| `waas test-results` | Show test results |
| `waas mock` | Run a local mock endpoint |
| `waas mock-endpoints` | List remote mock endpoints |
| `waas tunnel` | Create a local development tunnel for webhooks |

### Advanced

| Command | Description |
|---------|-------------|
| `waas generate` | Generate webhook config from an OpenAPI spec |
| `waas contracts` | Manage webhook contracts and run validation |
| `waas tenant` | Manage tenants |
| `waas migrate` | Migrate data from other webhook providers |
| `waas export` | Export data to a file |
| `waas import` | Import data from a file |

### GitOps

| Command | Description |
|---------|-------------|
| `waas apply` | Apply declarative configuration from YAML files |
| `waas diff` | Show diff between local config and remote state |
| `waas drift` | Detect configuration drift |

### Utility

| Command | Description |
|---------|-------------|
| `waas version` | Print CLI version |

## Usage Examples

### Quick Start

```bash
# Authenticate
waas login --api-key your-key-here

# Check connection
waas health

# List endpoints
waas endpoints list

# Send a webhook
waas send --endpoint <endpoint-id> --payload '{"event": "user.created", "data": {"id": "123"}}'
```

### Monitoring

```bash
# Stream logs in real-time
waas logs --tail

# Inspect a specific delivery
waas inspect <delivery-id>

# Check account usage
waas status
```

### Testing Webhooks Locally

```bash
# Start a local mock endpoint
waas mock --port 9000

# In another terminal, create a tunnel
waas tunnel --port 9000

# Send a test webhook
waas send --endpoint <endpoint-id> --payload '{"test": true}'
```

### GitOps Workflow

```bash
# Export current config
waas export --output config.yaml

# Preview changes
waas diff -f config.yaml

# Apply configuration
waas apply -f config.yaml

# Check for drift
waas drift
```

### Replaying Failed Deliveries

```bash
# Replay a single delivery
waas replay <delivery-id>

# Bulk replay all failed deliveries from the last hour
waas bulk-replay --since 1h --status failed
```

### Output Formats

```bash
# JSON output (useful for scripting)
waas endpoints list -o json

# YAML output
waas endpoints list -o yaml

# CSV output
waas endpoints list -o csv
```

### Shell Completion

```bash
# Bash
waas completion bash > /etc/bash_completion.d/waas

# Zsh
waas completion zsh > "${fpath[1]}/_waas"

# Fish
waas completion fish > ~/.config/fish/completions/waas.fish
```
