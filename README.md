# SWAT

Deploy your own AI army. SWAT dispatches autonomous squads powered by GitHub Copilot CLI — orchestrated through [OpenClaw](https://github.com/openclaw/openclaw).

## 🚀 Installation

```bash
curl -fsSL https://raw.githubusercontent.com/LangSensei/swat-v2/main/install.sh | bash
```

This will:
1. Download the latest release for your platform (Linux/macOS, amd64/arm64)
2. Install the SWAT binary to `~/.local/bin/` (auto-added to PATH in your shell profile)
3. Set up the OpenClaw bridge plugin at `~/.swat/plugin/`
4. Install framework blueprints to `~/.swat/blueprints/`
5. Auto-register the plugin in your OpenClaw config

Then restart OpenClaw and you're ready to go.

## 🗑️ Uninstallation

```bash
curl -fsSL https://raw.githubusercontent.com/LangSensei/swat-v2/main/uninstall.sh | bash
```

Add `--purge` to also remove runtime data (operation history, intel).

## 🎮 Usage

SWAT is controlled entirely through OpenClaw — no CLI needed. Just chat naturally, or use the 10 built-in tools:

### Operations
- `swat_dispatch` — Dispatch a task (auto-classified to the right squad)
- `swat_ops` — List operations with filters (status/since/limit/offset)
- `swat_cancel` — Cancel a running operation

### Squads
- `swat_squads` — List installed squads
- `swat_squad_browse` — Browse the marketplace
- `swat_squad_install` — Install a squad from the marketplace
- `swat_squad_uninstall` — Uninstall a squad
- `swat_squad_update` — Update a squad to the latest marketplace version

### Schedule
- `swat_schedule_create` — Create a recurring task (zero LLM cost)
- `swat_schedules` — List all schedules
- `swat_schedule_delete` — Delete a schedule

For detailed tool usage and monitoring patterns, see [`skill/SKILL.md`](skill/SKILL.md).

### CLI Flags

The `swat` binary supports the following flags:

| Flag | Description |
|------|-------------|
| `--version` | Print the installed version (`swat <version>`) and exit |
| `--mcp-only` | Start only the MCP server — skip the background commander loop |

## 🧠 Architecture

```
OpenClaw (HQ) → Bridge Plugin → SWAT Commander (Go MCP) → Squads (Copilot CLI)
```

1. **OpenClaw** — Your interface. Chat to dispatch and monitor tasks.
2. **Commander** — Go MCP server. Handles dispatch, workspace composition, dependency resolution, scheduling, and completion scanning.
3. **Squads** — Specialist agents. Each runs as an independent Copilot CLI process with its own skills, MCP tools, and protocol.

### How It Works

1. You dispatch a task (Commander auto-classifies to the right squad)
2. Commander provisions the workspace — copies squad snapshot to `.squad/`, skill hooks to `.github/hooks/`, skill content to `.github/skills/`, resolves MCP dependencies, writes AGENTS.md (protocol)
3. Copilot CLI launches in the operation directory, reads AGENTS.md + OPERATION.md
4. Commander's background loop scans for completion (OPERATION.md status + report.html)
5. Results surface through OpenClaw

### Scheduler

Built-in Go cron scheduler for recurring tasks — zero LLM cost:
- Standard 5-field cron expressions with timezone support
- `immediate` flag for first-run-now behavior
- In-flight protection (skips if previous run still active)
- Startup catch-up (checks due schedules on boot)

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
│   ├── _unclassified/             # Operations before squad assignment
│   └── <squad>/
│       ├── INTEL.md               # Persistent cross-operation knowledge
│       └── operations/
│           └── <id>/              # Operation workspace
│               ├── OPERATION.md
│               ├── AGENTS.md      # Protocol (copied from _framework/PROTOCOL.md)
│               ├── .mcp.json
│               ├── report.html
│               ├── .squad/        # Squad blueprint snapshot (read-only)
│               └── .github/
│                   ├── hooks/     # Skill hooks (Copilot CLI native)
│                   └── skills/    # Skill content (SKILL.md + templates)
│
├── schedules/                     # Schedule definitions (JSON)
│   └── <id>.json
│
└── plugin/                        # OpenClaw bridge plugin
```

## 🛠️ Prerequisites

- Linux or macOS
- [GitHub Copilot CLI](https://www.npmjs.com/package/@github/copilot) (`npm install -g @github/copilot`)
- Node.js 18+
- [OpenClaw](https://github.com/openclaw/openclaw)

## 🔨 Building from Source

```bash
go build -o swat .   # Go 1.24+
# With version injection:
go build -ldflags "-X main.version=v1.0.0" -o swat .
```

## 📄 License

MIT
