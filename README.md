# tgh

A terminal UI for managing GitHub pull requests. Browse, review, approve, merge, and close PRs across multiple repositories without leaving the terminal.

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and powered by the [GitHub CLI](https://cli.github.com/) (`gh`).

## Features

- View PRs needing your review or authored by you
- Approve, merge (squash), close, and update branches
- Bulk-select and operate on multiple PRs at once
- Filter by title, author, or repository
- Inline detail panel with description, CI checks, and review statuses
- Open PRs in the browser or copy URLs to clipboard
- Single batched GraphQL query for CI, review, and merge statuses

## Prerequisites

- [GitHub CLI](https://cli.github.com/) (`gh`) installed and authenticated

## Installation

```
go install github.com/pivovarit/tgh@latest
```

Or download a binary from the [releases page](https://github.com/pivovarit/tgh/releases).

## Usage

```
tgh [flags]
```

### Flags

| Flag | Description |
|------|-------------|
| `-owner` | Limit to a specific owner/org (repeatable) |

### Examples

```sh
tgh
tgh -owner pivovarit
tgh -owner pivovarit -owner vavr-io
```