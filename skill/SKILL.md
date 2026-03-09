# SWAT - Autonomous Squad Orchestration

SWAT dispatches tasks to autonomous AI squads powered by GitHub Copilot CLI. Each squad is a domain specialist that works independently in the background.

## CLI

| Command | Purpose |
|---|---|
| `swat --version` | Show installed version |

## Tools

### Operations
| Tool | Purpose |
|---|---|
| `swat_dispatch` | Dispatch a task (auto-classified to the right squad) |
| `swat_ops` | List operations (supports status/since/limit/offset filters), returns counts |
| `swat_cancel` | Cancel a running operation |

### Squads
| Tool | Purpose |
|---|---|
| `swat_squads` | List installed squads |
| `swat_squad_browse` | List squads available in the marketplace |
| `swat_squad_install` | Install a squad from the marketplace |
| `swat_squad_uninstall` | Uninstall a squad and clean up dependencies |
| `swat_squad_update` | Update an installed squad to the latest marketplace version |

### Schedule
| Tool | Purpose |
|---|---|
| `swat_schedule_create` | Create a scheduled recurring task (zero LLM cost) |
| `swat_schedules` | List all scheduled tasks with next run times |
| `swat_schedule_delete` | Delete a scheduled task |

## How to Dispatch

1. **Dispatch** — `swat_dispatch(brief)`. Squad is auto-classified. Returns an operation ID immediately.
2. **Tell the user** — Confirm the task is dispatched. Classification + launch happens async in the background.
3. **Move on** — Do NOT wait, poll, or sleep. The squad works in the background.

## Scheduling Recurring Tasks

Use SWAT's built-in scheduler for deterministic, zero-LLM-cost recurring tasks:

```
swat_schedule_create(brief="分析紫金矿业601899", cron="0 9 * * 1", timezone="Asia/Shanghai")
```

- **Cron format**: Standard 5-field — `min hour dom month dow`
- **`immediate`**: Set to `true` to trigger the first run right away (default: `false`)
- **In-flight protection**: If a previous run from the same schedule is still queued/active, the next trigger is skipped
- **Startup catch-up**: Due schedules are checked on SWAT startup — no missed runs after restarts
- **Source tracking**: Operations from schedules have `source: schedule/{id}` for traceability

Use `swat_schedules` to view all schedules and `swat_schedule_delete(id)` to remove one.

**When to use SWAT scheduler vs OpenClaw cron:**
- SWAT scheduler → deterministic recurring tasks (zero LLM cost, e.g. "analyze X every Monday")
- OpenClaw cron → tasks needing LLM judgment (e.g. completion monitoring with active-diff)

## Checking Results

- Call `swat_ops` **only when the user asks** about a task, or when you have a natural reason to check (e.g., heartbeat).
- **Never** use `sleep`, polling loops, or repeated `exec` calls to wait for completion. This blocks the main session and makes you unresponsive.
- `swat_ops` returns counts + operations. Filters:
  - `status` — `queued`, `active`, `completed`, `failed`
  - `since` — RFC3339 timestamp (e.g. `2026-03-09T04:00:00Z`), only returns terminal ops after this time; active/queued always included
  - `limit` — max results (default 50)
  - `offset` — skip first N results (default 0)
  - Results sorted by time descending (most recent first)

## Completion Monitoring (Active-Diff Pattern)

After dispatching one or more tasks, set up a **cron job** to detect completions via active-list diffing:

```
cron(action=add, job={
  name: "swat-monitor",
  schedule: { kind: "every", everyMs: 120000 },
  sessionTarget: "isolated",
  payload: {
    kind: "agentTurn",
    message: "You are a SWAT completion monitor. Follow these steps exactly:\n\n1. Read workspace file memory/swat-monitor.json. If it doesn't exist, treat lastActiveIds as [].\n2. Call swat_ops(status=active) to get current active operation IDs.\n3. Compute disappeared = IDs in lastActiveIds that are NOT in current active IDs.\n4. For each disappeared ID, call swat_list to find its details (it will be completed or failed). Send a summary to the user (operation ID, squad, brief, status, summary).\n5. Write memory/swat-monitor.json with lastActiveIds = current active IDs.\n6. If current active count is 0 AND no disappeared IDs, delete this cron job.\n7. If nothing to report, reply NO_REPLY."
  },
  delivery: { mode: "announce" }
})
```

- **Auto-delete**: When active=0 and no new completions detected, the cron deletes itself.
- **Don't stack**: Only create one monitor cron at a time. Check if one exists before creating another.
- **Interval**: 2 minutes is a good default. Adjust if the user wants faster/slower updates.

## Critical Rules

1. **Fire and forget** — After dispatch, immediately return control to the user. Do not monitor, poll, or block.
2. **No sleep/exec polling** — Never run `sleep X && check` or similar patterns. SWAT tasks can take minutes; blocking the session makes you unreachable.
3. **Auto-classification** — You do NOT need to pick a squad. `swat_dispatch` auto-classifies the task. If no squad fits, the operation fails with a clear reason.
4. **Concurrent operations** — Multiple tasks can run in parallel across different squads.
5. **Failed operations** — Include the failure reason when reporting to the user. Unclassified failures stay in `_unclassified/`.

## Marketplace

- `swat_squad_browse` — See what's available to install (fetches from GitHub, no clone needed).
- `swat_squad_install(squad)` — Downloads squad + resolves dependencies automatically.
- `swat_squad_uninstall(squad)` — Removes squad blueprint + cleans up orphaned dependencies.

## First Run

If SWAT tools are not available, guide the user to install:
```
curl -fsSL https://raw.githubusercontent.com/LangSensei/swat-v2/master/install.sh | bash
```
Then restart OpenClaw. After that, install a squad: `swat_squad_install("squad-name")`.

Before the first dispatch, verify GitHub auth is set up (required for Copilot CLI):
```bash
gh auth status
```
If not authenticated, guide the user to run `gh auth login` first.
