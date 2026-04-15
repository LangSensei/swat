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
2. Install the SWAT binary to `~/.swat/bin/` and add it to your PATH (Linux/macOS: appends to `.bashrc`/`.zshrc`; Windows: updates user PATH in the registry)
3. Install framework blueprints to `~/.swat/blueprints/`

### OpenClaw Integration (Optional)

If you use [OpenClaw](https://github.com/openclaw/openclaw), install the bridge plugin separately:

```bash
# See https://github.com/LangSensei/swat-openclaw
```

## 🗑️ Uninstallation

### Linux / macOS

```bash
curl -fsSL https://raw.githubusercontent.com/LangSensei/swat/main/uninstall.sh | bash
```

### Windows

```powershell
irm https://raw.githubusercontent.com/LangSensei/swat/main/uninstall.ps1 | iex
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

SWAT is **runtime-agnostic**. The Commander orchestrates operations while a pluggable **RuntimeAdapter** handles runtime-specific details (dot-directories, agent files, MCP config paths, hooks).

| Component | Role |
|-----------|------|
| **Commander** | Go MCP server. Handles dispatch, workspace composition, dependency resolution, scheduling, and completion scanning. |
| **Squads** | Specialist agents. Each runs as an independent coding agent session with its own skills, MCP tools, and protocol. |
| **RuntimeAdapter** | Abstracts runtime differences. Each adapter knows its dot-dir, agent file name, MCP config path, and how to launch the agent. |

### Supported Runtimes

| Runtime | Dot-Dir | Agent File | MCP Config | Launch Command |
|---------|---------|------------|------------|----------------|
| `copilot` (default) | `.github/` | `AGENTS.md` | `.mcp.json` | `copilot` |
| `gemini` | `.gemini/` | `GEMINI.md` | `.gemini/settings.json` | `gemini` |

Set the runtime with `--runtime <name>` (e.g. `swat --runtime gemini`).

### How It Works

1. You dispatch a task (Commander auto-classifies to the right squad)
2. Commander provisions the workspace — copies squad snapshot to `.squad/`, skill hooks to `<dotdir>/hooks/`, skill content to `<dotdir>/skills/`, resolves MCP dependencies, writes the agent file (runtime-specific)
3. A coding agent session launches in the operation directory, reads the agent file + OPERATION.md
4. Commander's background loop scans for completion (OPERATION.md status + report.html)
5. Results surface through your agent

> **Skills vs Hooks:** Skills are runtime-agnostic (shared across all runtimes). Hooks are runtime-specific and live under `hooks/copilot/`, `hooks/gemini/`, etc. in each skill's blueprint.

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
│   ├── skills/                    # Shared skills (runtime-agnostic)
│   └── mcps/                      # MCP server configs
│
├── squads/                        # Runtime data
│   ├── _unclassified/             # Operations before squad assignment
│   └── <squad>/
│       └── operations/
│           └── <id>/              # Operation workspace
│               ├── OPERATION.md
│               ├── <agent-file>   # Runtime-specific (AGENTS.md or GEMINI.md)
│               ├── <mcp-config>   # Runtime-specific (.mcp.json or .gemini/settings.json)
│               ├── report.html
│               ├── .squad/        # Squad blueprint snapshot (read-only)
│               └── <dotdir>/      # Runtime-specific (.github/ or .gemini/)
│                   ├── hooks/     # Skill hooks (runtime-specific)
│                   └── skills/    # Skill content
│
└── schedules/                     # Schedule definitions (JSON)
    └── <id>.json
```

## 🛠️ Prerequisites

- Linux, macOS, or Windows
- [GitHub CLI](https://cli.github.com) (authenticated)
- At least one supported AI coding agent runtime:

| Runtime | Requirements |
|---------|-------------|
| **Copilot** (default) | [GitHub Copilot CLI](https://www.npmjs.com/package/@github/copilot) (`npm install -g @github/copilot`), Node.js 18+ |
| **Gemini** | [Gemini CLI](https://github.com/google-gemini/gemini-cli) installed and available as `gemini` on PATH |

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
