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

## Usage

```sh
itui
```

The TUI reads credentials from `~/.imitsu/config.json`, shared with the CLI. If you've already logged in via `imitsu login`, the TUI picks up the session automatically.

## Keybindings

| Key | Action |
|---|---|
| `s` / `t` | Switch between secrets and teams tabs |
| `j` / `k` | Navigate up/down |
| `enter` | View detail |
| `n` | New secret (in secrets tab) |
| `d` | Delete secret (in detail view) |
| `r` | Refresh |
| `esc` | Go back |
| `ctrl+r` | Toggle login/register |
| `ctrl+l` | Logout |
| `q` | Quit |

## Release

Pushing a tag like `v0.1.0` triggers a GitHub Actions workflow that cross-compiles binaries for macOS and Linux (amd64 + arm64) via GoReleaser.

```sh
git tag v0.1.0
git push origin v0.1.0
```
