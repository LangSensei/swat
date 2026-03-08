---
name: intel
version: "1.0.0"
description: Squad-level persistent intel template
---

# {SQUAD_NAME} Squad — Intel

> This file persists across operations. Update it at seal time.
> - Add a row to the Operation Log for every completed operation
> - Update Persistent Insights with cross-operation patterns
> - Keep insights concise and actionable — this is NOT a dump of findings

## Operation Log

> One row per completed operation. Most recent first.
> Tags are freeform — use short, lowercase labels for fast lookup.

| Operation | Date | Tags | Brief | Outcome |
|-----------|------|------|-------|---------|

## Persistent Insights

> Cross-operation patterns, recurring issues, and squad-level learnings.
> Only add insights that would help future operations.
> Remove or update insights that become stale.
> Prefix each insight with a hit count `(×N)` — increment N each time a new operation confirms the pattern.

- **(×1) Example insight title:** description of the pattern...
