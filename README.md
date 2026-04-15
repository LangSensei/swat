# SWAT

Autonomous multi-agent task execution engine. SWAT dispatches tasks to specialist squads — each running as an independent AI coding agent session.

## 🚀 Installation

### Linux / macOS

```bash
curl -fsSL https://raw.githubusercontent.com/LangSensei/swat/main/install.sh | bash
```

### Windows

```powershell
irm https://raw.githubusercontent.com/LangSensei/swat/main/install.ps1 | iex
```

This will:
1. Download the latest release for your platform
2. Install the SWAT binary to `~/.swat/bin/` with a symlink at `~/.local/bin/swat` (Linux only; auto-added to PATH)
3. Install framework blueprints to `~/.swat/blueprints/`

### OpenClaw Integration (Optional)

If you use [OpenClaw](https://github.com/openclaw/openclaw), install the bridge plugin separately:

```bash
# See https://github.com/LangSensei/swat-openclaw
```

## 🗑️ Uninstallation

```bash
curl -fsSL https://raw.githubusercontent.com/LangSensei/swat/main/uninstall.sh | bash
```

Add `--purge` to also remove runtime data (operation history).

## 🎮 Usage

SWAT exposes 12 MCP tools. Configure your agent to connect:

### MCP Configuration

```json
{
  "mcpServers": {
    "swat": {
      "command": "swat",
      "args": []
    }
  }
}
```

### Tools

#### Operations
- `swat_dispatch` — Dispatch a task (auto-classified to the right squad)
- `swat_ops` — List operations with filters (status/since/limit/offset)
- `swat_cancel` — Cancel a running operation

#### Squads
- `swat_squads` — List installed squads
- `swat_squad_browse` — Browse the marketplace
- `swat_squad_install` — Install a squad from the marketplace
- `swat_squad_uninstall` — Uninstall a squad
- `swat_squad_update` — Update a squad to the latest marketplace version

#### Schedule
- `swat_schedule_create` — Create a recurring task (zero LLM cost)
- `swat_schedules` — List all schedules
- `swat_schedule_delete` — Delete a schedule

#### Notification
- `swat_notify` — Send a notification to the user

### CLI Flags

| Flag | Description |
|------|-------------|
| `--version` | Print the installed version and exit |
| `--mcp-only` | Start only the MCP server — skip the background commander loop |
| `--runtime <name>` | Set the AI coding agent runtime (default: `copilot`) |
| `--notify <target>` | Set the notification target: `desktop`, `openclaw` (default: `desktop`) |

## 🧠 Architecture

```
Agent → MCP → SWAT Commander (Go) → Squads (AI coding agent sessions)
```

1. **Commander** — Go MCP server. Handles dispatch, workspace composition, dependency resolution, scheduling, and completion scanning.
2. **Squads** — Specialist agents. Each runs as an independent coding agent session with its own skills, MCP tools, and protocol.

### How It Works

1. You dispatch a task (Commander auto-classifies to the right squad)
2. Commander provisions the workspace — copies squad snapshot to `.squad/`, skill hooks to `.github/hooks/`, skill content to `.github/skills/`, resolves MCP dependencies, writes AGENTS.md (protocol)
3. A coding agent session launches in the operation directory, reads AGENTS.md + OPERATION.md
4. Commander's background loop scans for completion (OPERATION.md status + report.html)
5. Results surface through your agent

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
│   │   │   └── TEMPLATE.md
│   │   └── <squad>/               # Squad blueprints
│   │       └── MANIFEST.md
│   ├── skills/                    # Shared skills
│   └── mcps/                      # MCP server configs
│
├── squads/                        # Runtime data
│   ├── _unclassified/             # Operations before squad assignment
│   └── <squad>/
│       └── operations/
│           └── <id>/              # Operation workspace
│               ├── OPERATION.md
│               ├── AGENTS.md      # Protocol (copied from _framework/PROTOCOL.md)
│               ├── .mcp.json
│               ├── report.html
│               ├── .squad/        # Squad blueprint snapshot (read-only)
│               └── .github/
│                   ├── hooks/     # Skill hooks
│                   └── skills/    # Skill content
│
└── schedules/                     # Schedule definitions (JSON)
    └── <id>.json
```

## 🛠️ Prerequisites

- Linux, macOS, or Windows
- [GitHub Copilot CLI](https://www.npmjs.com/package/@github/copilot) (`npm install -g @github/copilot`)
- Node.js 18+
- [GitHub CLI](https://cli.github.com) (authenticated)

## 🔨 Building from Source

```bash
go build -o swat .   # Go 1.24+
# With version injection:
go build -ldflags "-X main.version=v1.0.0" -o swat .
```

## 📦 Related Projects

- [swat-marketplace](https://github.com/LangSensei/swat-marketplace) — Squads, skills, and MCPs
- [swat-openclaw](https://github.com/LangSensei/swat-openclaw) — OpenClaw integration (plugin + skill)

## 📄 License

MIT
