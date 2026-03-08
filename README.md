# SWAT v2

AI task orchestration system. Deploy autonomous squads to handle your tasks.

## Architecture

```
OpenClaw (HQ) → MCP Bridge Plugin → SWAT Commander (Go) → Squad (Copilot CLI)
```

- **Commander**: Go MCP server — task queue, classification, scheduling, review
- **Plugin**: OpenClaw bridge — spawns Commander, registers native tools
- **Squad**: Independent Copilot CLI processes — autonomous task execution

## Tools

| Tool | Description |
|---|---|
| swat_dispatch | Dispatch a new task to a squad |
| swat_status | Get task status + unnotified completions |
| swat_list | List all operations |
| swat_cancel | Cancel an operation |
| swat_squads | List installed squads |
| swat_schedule | Create a scheduled task |

## Build

```bash
go build -o swat .
```

## Install Plugin

```bash
openclaw plugins install ./plugin
```
