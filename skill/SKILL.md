# SWAT - Autonomous Squad Orchestration

SWAT dispatches tasks to autonomous AI squads powered by GitHub Copilot CLI. Each squad is a domain specialist that works independently in the background.

## Tools

| Tool | Purpose |
|---|---|
| `swat_dispatch` | Send a task to a squad |
| `swat_status` | Check for completions and unnotified results |
| `swat_squads` | List installed squads |
| `swat_list` | List all operations with status |
| `swat_cancel` | Cancel a running operation |
| `swat_schedule` | Create a scheduled/recurring task |
| `swat_browse` | List squads available in the marketplace |
| `swat_install` | Install a squad from the marketplace |
| `swat_uninstall` | Uninstall a squad and clean up dependencies |

## How to Dispatch

1. **Pick a squad** — Call `swat_squads` to see installed squads. If none fit, use `swat_browse` + `swat_install`.
2. **Dispatch** — `swat_dispatch(brief, squad)`. Returns an operation ID immediately.
3. **Tell the user** — Confirm the task is dispatched and which squad is on it.
4. **Move on** — Do NOT wait, poll, or sleep. The squad works in the background.

## Checking Results

- Call `swat_status` **only when the user asks** about a task, or when you have a natural reason to check (e.g., heartbeat).
- **Never** use `sleep`, polling loops, or repeated `exec` calls to wait for completion. This blocks the main session and makes you unresponsive.
- `swat_status` returns unnotified completions. Summarize the result to the user.
- `swat_list` shows all operations if you need the full picture.

## Completion Monitoring

After dispatching one or more tasks, set up a **cron job** to poll for completions:

```
cron(action=add, job={
  name: "swat-monitor",
  schedule: { kind: "every", everyMs: 120000 },
  sessionTarget: "isolated",
  payload: {
    kind: "agentTurn",
    message: "Call swat_status. If there are unnotified results, summarize each one (operation ID, squad, brief, status, summary) and send to the user. Then call swat_list to mark them seen. If no unnotified results, reply NO_REPLY."
  },
  delivery: { mode: "announce" }
})
```

- **Auto-delete**: When all dispatched tasks are done (no active operations), delete the cron job.
- **Don't stack**: Only create one monitor cron at a time. Check if one exists before creating another.
- **Interval**: 2 minutes is a good default. Adjust if the user wants faster/slower updates.

## Critical Rules

1. **Fire and forget** — After dispatch, immediately return control to the user. Do not monitor, poll, or block.
2. **No sleep/exec polling** — Never run `sleep X && check` or similar patterns. SWAT tasks can take minutes; blocking the session makes you unreachable.
3. **Squad selection matters** — Pick the squad whose domain matches the task. If no squad fits, say so.
4. **Concurrent operations** — Multiple tasks can run in parallel across different squads.
5. **Failed operations** — Include the failure reason when reporting to the user.

## Marketplace

- `swat_browse` — See what's available to install (fetches from GitHub, no clone needed).
- `swat_install(squad)` — Downloads squad + resolves dependencies automatically.
- `swat_uninstall(squad)` — Removes squad blueprint + cleans up orphaned dependencies.

## First Run

If SWAT tools are not available, guide the user to install:
```
curl -fsSL https://raw.githubusercontent.com/LangSensei/swat-v2/master/install.sh | bash
```
Then restart OpenClaw. After that, install a squad: `swat_install("squad-name")`.
