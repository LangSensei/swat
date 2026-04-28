---
# ── COMMANDER (written at dispatch, do not modify) ──
# format: YYYYMMDD-8hex
operation_id: {OPERATION_ID}
pid:
created_at: {CREATED_AT}
dispatched_at:
failed_at:
failure_reason:

# ── CLASSIFY (written by LLM at classify, do not modify) ──
squad:
# e.g., [{type: "operation", value: "../20260309-xxxx/"}]
references: []

# ── OPERATOR (fill during execution) ──
# queued → active → completed / failed
status: queued
# 2-3 sentence summary of outcome
summary:
completed_at:
---

# {BRIEF}

## Assignment
{DETAILS}

### Context
[CLASSIFY: Historical context, related operations, key metrics from past runs]
