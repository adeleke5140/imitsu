# itui

An interactive terminal UI for [imitsu](../../README.md), built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/adeleke5140/imitsu/main/install.sh | sh
```

Or with Go:

```sh
go install github.com/adeleke5140/imitsu/tui@latest
```

Or build from source:

```sh
cd src/tui
go build -o itui .
```

## Getting Started

```sh
itui
```

On first launch you'll land on the **account** tab. From there:

1. Go to **server** in the sidebar and set your imitsu server URL
2. Go to **register** to create an account, or **login** if you already have one
3. After logging in you'll be switched to the secrets tab

Configuration is stored in `~/.imitsu/config.json` and shared with the CLI. If you've already run `imitsu server <url>` and `imitsu login`, the TUI picks up the session automatically.

## Tabs

| Key | Tab |
|---|---|
| `s` | Secrets — list, view, create, delete, export |
| `t` | Teams — list teams, view members |
| `a` | Account — login, register, server config, profile |

## Keybindings

### Global

| Key | Action |
|---|---|
| `s` / `t` / `a` | Switch tabs |
| `q` | Quit |
| `ctrl+l` | Logout |

### Secrets

| Key | Action |
|---|---|
| `j` / `k` | Navigate up/down |
| `enter` | View secret detail |
| `n` | New secret |
| `e` | Export secrets to .env file |
| `d` | Delete (in detail view) |
| `r` | Refresh |
| `esc` | Go back |

### Account sidebar

| Key | Action |
|---|---|
| `j` / `k` | Navigate sidebar |
| `enter` | Select pane |
| `tab` | Next form field |
| `esc` | Back to sidebar |

## Release

Pushing a tag like `v0.1.0` triggers a GitHub Actions workflow that cross-compiles binaries for macOS and Linux (amd64 + arm64) via GoReleaser.

```sh
git tag v0.1.0
git push origin v0.1.0
```
