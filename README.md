# SWAT

Deploy your own AI army. SWAT integrates with GitHub Copilot CLI to dispatch autonomous squads that handle your development and operational tasks — orchestrated through [OpenClaw](https://github.com/openclaw/openclaw).

## 🚀 Installation

```bash
curl -fsSL https://raw.githubusercontent.com/LangSensei/swat-v2/master/install.sh | bash
```

This will:
1. Download the latest release for your platform (Linux/macOS, amd64/arm64)
2. Install the SWAT binary to `~/.local/bin/`
3. Set up the OpenClaw bridge plugin at `~/.swat/plugin/`
4. Install framework blueprints to `~/.swat/blueprints/`

Then register the plugin in your OpenClaw config and restart:
```json
{ "plugins": ["~/.swat/plugin"] }
```

## 🎮 Usage

SWAT is controlled entirely through OpenClaw — no CLI commands needed. Just chat:

> "Create a Python script that calculates fibonacci numbers"

Or use the tools directly:

| Tool | Description |
|---|---|
| `swat_dispatch` | Dispatch a new task to a squad |
| `swat_status` | Get task status and unnotified completions |
| `swat_list` | List all operations |
| `swat_cancel` | Cancel an operation |
| `swat_squads` | List installed squads |
| `swat_schedule` | Create a scheduled task |

### Install Squads

Install specialized squads from the [SWAT Marketplace](https://github.com/LangSensei/swat-marketplace):

```bash
# Copy squad blueprints to ~/.swat/blueprints/squads/
# Copy skills to ~/.swat/blueprints/skills/
# Copy MCP configs to ~/.swat/blueprints/mcps/
```

## 🧠 Architecture

```
OpenClaw (HQ) → Bridge Plugin → SWAT Commander (Go) → Squads (Copilot CLI)
```

1. **OpenClaw (HQ)** — Your primary interface. Chat to plan and assign tasks.
2. **Commander** — Go MCP server that orchestrates the task lifecycle: dispatch, compose, provision, scan, and review.
3. **Squads** — Specialized agent configurations. Each squad runs as an independent GitHub Copilot CLI process with its own tools (skills & MCPs), knowledge (intel), and protocol.

### How It Works

1. You dispatch a task (with or without specifying a squad)
2. Commander composes the workspace — assembles AGENTS.md from protocol + manifest, resolves skill dependencies recursively, generates `.mcp.json`
3. A Copilot CLI process launches in the operation directory, reads OPERATION.md for the task brief, follows the protocol in AGENTS.md
4. Commander periodically scans for completion (checks OPERATION.md status + report.html)
5. Results are reported back to OpenClaw

## 📂 Directory Structure

```
~/.swat/
├── blueprints/                    # Templates & marketplace content
│   ├── OPERATION.md               # Operation template
│   ├── squads/
│   │   ├── _framework/            # Core protocol (versioned with binary)
│   │   │   ├── PROTOCOL.md
│   │   │   ├── TEMPLATE.md
│   │   │   └── INTEL.md
│   │   └── coding/                # Squad blueprints (from marketplace)
│   │       └── MANIFEST.md
│   ├── skills/                    # Shared skills (from marketplace)
│   └── mcps/                      # MCP server configs (from marketplace)
│
├── squads/                        # Runtime data
│   └── coding/
│       ├── INTEL.md               # Persistent cross-operation experience
│       └── operations/
│           └── 20260308-a1b2c3d4/ # Operation workspace
│               ├── OPERATION.md   # Task brief (single source of truth)
│               ├── AGENTS.md      # Assembled protocol + manifest
│               ├── .mcp.json      # Composed MCP config
│               ├── report.html    # Completion report
│               └── .github/skills/
│
└── plugin/                        # OpenClaw bridge plugin
```

## 🛠️ Prerequisites

- Linux or macOS
- [GitHub Copilot CLI](https://githubnext.com/projects/copilot-cli) (`copilot`)
- Node.js 18+ (for plugin and MCP servers)
- [OpenClaw](https://github.com/openclaw/openclaw)

## 🔨 Building from Source

```bash
# Requires Go 1.24+
go build -o swat .
```

## 📄 License

MIT
