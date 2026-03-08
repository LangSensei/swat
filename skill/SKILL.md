# SWAT - Autonomous Squad Orchestration

SWAT dispatches tasks to autonomous AI squads powered by GitHub Copilot CLI. Each squad is a specialist (coding, research, etc.) that works independently in the background.

## When to Use

- User wants a task done autonomously in the background
- Tasks that benefit from a specialist agent (coding, research, writing)
- Multiple tasks that can run in parallel
- User says things like "do this for me", "handle this", "work on this in the background"

## Tools

| Tool | Purpose |
|---|---|
| `swat_dispatch` | Send a task to a squad |
| `swat_status` | Check for completions and unnotified results |
| `swat_squads` | List available squads |
| `swat_list` | List all operations with status |
| `swat_cancel` | Cancel a running operation |
| `swat_schedule` | Create a scheduled/recurring task |

## How to Use

### 1. Check available squads
Call `swat_squads` to see what's installed. Each squad has a domain (e.g., coding, research).

### 2. Dispatch a task
```
swat_dispatch:
  brief: "Short task description" (required)
  squad: "squad-name" (required — must match an installed squad)
  details: "Additional context, requirements, links" (optional)
```

### 3. Monitor progress
- `swat_status` — Returns unnotified completions. Call after dispatch or periodically.
- `swat_list` — Shows all operations (queued, active, completed, failed).

### 4. Report results
When `swat_status` returns a completed operation, summarize the result to the user.

### 5. Cancel if needed
```
swat_cancel:
  operation_id: "20260308-a1b2c3d4"
```

## Typical Flow

1. User requests a task → check `swat_squads` if unsure which squad
2. `swat_dispatch` with brief + squad → returns operation ID immediately
3. Tell the user the task is dispatched and which squad is handling it
4. Later, `swat_status` picks up the completion → summarize to user

## Important Notes

- **Dispatch is async** — the squad works in the background, don't wait for it
- **Squad selection matters** — pick the squad whose domain matches the task
- **If no squad fits** — tell the user; don't dispatch to a random squad
- **Multiple operations** can run concurrently across different squads
- **Results** are in OPERATION.md when status becomes `completed`
- **Failed operations** include a failure reason in the status response
