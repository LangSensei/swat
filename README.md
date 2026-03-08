# SWAT

Deploy your own AI army. SWAT dispatches autonomous squads powered by GitHub Copilot CLI — orchestrated through [OpenClaw](https://github.com/openclaw/openclaw).

## 🚀 Installation

```bash
curl -fsSL https://raw.githubusercontent.com/LangSensei/swat-v2/master/install.sh | bash
```

This will:
1. Download the latest release for your platform (Linux/macOS, amd64/arm64)
2. Install the SWAT binary to `~/.local/bin/`
3. Set up the OpenClaw bridge plugin at `~/.swat/plugin/`
4. Install framework blueprints to `~/.swat/blueprints/`
5. Auto-register the plugin in your OpenClaw config

Then restart OpenClaw and install a squad:
```
swat_browse    → see available squads
swat_install   → install one
swat_dispatch  → start a task
```

## 🗑️ Uninstallation

```bash
curl -fsSL https://raw.githubusercontent.com/LangSensei/swat-v2/master/uninstall.sh | bash
```

Add `--purge` to also remove runtime data (operation history, intel).

## 🎮 Usage

SWAT is controlled entirely through OpenClaw — no CLI needed. Just chat:

> "Analyze 贵州茅台's recent stock performance"

Or use the tools directly:

| Tool | Description |
|---|---|
| `swat_dispatch` | Dispatch a task to a squad |
| `swat_status` | Check for completions |
| `swat_list` | List all operations |
| `swat_cancel` | Cancel an operation |
| `swat_squads` | List installed squads |
| `swat_schedule` | Create a scheduled task |
| `swat_browse` | Browse squads in the marketplace |
| `swat_install` | Install a squad + dependencies |
| `swat_uninstall` | Uninstall a squad + clean orphans |

### Marketplace

Install specialized squads from the [SWAT Marketplace](https://github.com/LangSensei/swat-marketplace):

```
swat_browse                    → list available squads
swat_install(squad: "coding")  → install squad + resolve dependencies
swat_uninstall(squad: "coding") → remove squad + clean orphaned deps
```

No `git clone` needed — SWAT fetches directly from GitHub API.

## 🧠 Architecture

```
OpenClaw (HQ) → Bridge Plugin → SWAT Commander (Go MCP) → Squads (Copilot CLI)
```

1. **OpenClaw** — Your interface. Chat to dispatch and monitor tasks.
2. **Commander** — Go MCP server. Handles dispatch, workspace composition, dependency resolution, and completion scanning.
3. **Squads** — Specialist agents. Each runs as an independent Copilot CLI process with its own skills, MCP tools, and protocol.

### How It Works

1. You dispatch a task (specify squad or let Commander classify)
2. Commander composes the workspace — assembles AGENTS.md, resolves skill dependencies (BFS), generates `.mcp.json`
3. Copilot CLI launches in the operation directory, reads OPERATION.md + AGENTS.md
4. Commander's background loop scans for completion (OPERATION.md status + report.html)
5. Results surface through `swat_status`

## 📂 Directory Structure

```
~/.swat/
├── blueprints/                    # Templates & installed content
│   ├── OPERATION.md               # Operation template
│   ├── squads/
│   │   ├── _framework/            # Core protocol (versioned with binary)
│   │   │   ├── PROTOCOL.md
│   │   │   ├── TEMPLATE.md
│   │   │   └── INTEL.md
│   │   └── <squad>/               # Squad blueprints
│   │       └── MANIFEST.md
│   ├── skills/                    # Shared skills
│   └── mcps/                      # MCP server configs
│
├── squads/                        # Runtime data
│   └── <squad>/
│       ├── INTEL.md               # Persistent cross-operation knowledge
│       └── operations/
│           └── <id>/              # Operation workspace
│               ├── OPERATION.md
│               ├── AGENTS.md
│               ├── .mcp.json
│               ├── report.html
│               └── .github/skills/
│
└── plugin/                        # OpenClaw bridge plugin
```

## 🛠️ Prerequisites

- Linux or macOS
- [GitHub Copilot CLI](https://githubnext.com/projects/copilot-cli) (`copilot`)
- Node.js 18+
- [OpenClaw](https://github.com/openclaw/openclaw)

## 🔨 Building from Source

```bash
go build -o swat .   # Go 1.24+
```

## 📄 License

MIT
