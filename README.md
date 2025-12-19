# CCC - Claude Code Configuration Manager

A cross-platform CLI tool for managing Claude Code configurations (All code writing with vibe coding).

## ScreenShot

![CCC ScreenShot](https://github.com/linauror/ccc/blob/main/ScreenShot.png)

## Features

- Add, update, delete, and list Claude Code configurations
- Activate/deactivate configurations
- Automatic activation of first configuration
- Prevents deletion of active configurations
- Cross-platform support (Windows, Linux, macOS)
- Secure API key display (masked in listings)

## Installation

1. Clone or download the source code
2. Build the executable:

```bash
go build -o ccc main.go
```

3. Move the executable to your desired location

## Usage

### List all configurations

```bash
ccc list
# or
ccc ls
```

### Add a new configuration

```bash
ccc add -n "config-name" -u "https://api.anthropic.com" -k "your-api-key"
```

### Update an existing configuration

```bash
# Update only base URL
ccc update -n "config-name" -u "https://new-api-url.com"

# Update only API key
ccc update -n "config-name" -k "new-api-key"

# Update both fields
ccc update -n "config-name" -u "https://new-api-url.com" -k "new-api-key"
```

### Activate a configuration

```bash
ccc activate -n "config-name"
```

This command will activate the configuration and automatically apply the settings for Claude Code:

**On Windows:**

- Set `ANTHROPIC_BASE_URL` and `ANTHROPIC_AUTH_TOKEN` environment variables
- Set these environment variables permanently using `setx` command

**On Linux and macOS:**

- Create/update `~/.claude/settings.json` file with environment variables
- The settings file format:

```json
{
  "env": {
    "ANTHROPIC_AUTH_TOKEN": "your-api-key",
    "ANTHROPIC_BASE_URL": "your-base-url",
    "API_TIMEOUT_MS": "3000000",
    "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": 1
  }
}
```

- Claude Code will read these settings directly from the file

### Delete a configuration

```bash
ccc delete -n "config-name"
```

**Note:** You cannot delete the currently active configuration. You must activate another configuration first.

## Configuration File

The configurations are stored in `ccc-config.json` in the same directory as the executable. The file format is:

```json
{
  "configurations": [
    {
      "name": "default",
      "base_url": "https://api.anthropic.com",
      "api_key": "sk-ant-api03-...",
      "active": true
    },
    {
      "name": "backup",
      "base_url": "https://backup-api.com",
      "api_key": "sk-backup-...",
      "active": false
    }
  ]
}
```

## Security

- API keys are masked in the list view (only first 4 and last 4 characters are shown)
- Configuration file is stored with read/write permissions for owner only (0644)
- No sensitive information is logged or exposed in error messages

## Cross-Platform Compatibility

This tool works on:

- Windows
- Linux
- macOS

The configuration file is always created in the same directory as the executable, making it portable across different systems.

## Auto-Import on First Run

When CCC runs for the first time (no ccc-config.json exists), it will automatically import existing configurations:

**On Windows:**

- Reads `ANTHROPIC_BASE_URL` and `ANTHROPIC_AUTH_TOKEN` environment variables
- Extracts configuration name from the domain (e.g., `api.anthropic.com` → `anthropic`)

**On Linux and macOS:**

- Reads `~/.claude/settings.json` file
- Extracts configuration name from the `ANTHROPIC_BASE_URL` domain
- Preserves all existing settings while creating the CCC configuration

If no existing configuration is found, CCC starts with an empty configuration list, waiting for user input.

## Requirements

- Go 1.21 or later
- Uses only Go standard library, no external dependencies
