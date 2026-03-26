---
# ── COMMANDER (written at dispatch, do not modify) ──
# format: YYYYMMDD-8hex
operation_id: {OPERATION_ID}
source: {SOURCE}
pid: {PID}
created_at: {CREATED_AT}
dispatched_at: {DISPATCHED_AT}
failed_at: {FAILED_AT}
failure_reason: {FAILURE_REASON}

# ── CLASSIFY (written by LLM at classify, do not modify) ──
squad: {SQUAD}
# e.g., [{type: "operation", value: "../20260309-xxxx/"}]
references: {REFERENCES}

# ── CAPTAIN (fill during execution) ──
# queued → active → completed / failed
status: {STATUS}
# 2-3 sentence summary of outcome
summary: {SUMMARY}
completed_at: {COMPLETED_AT}
# {OUTPUT_SCHEMA}
---

# {BRIEF}

## Assignment
{DETAILS}

### Context
[CLASSIFY: Historical context, related operations, key metrics from past runs]

## Findings
[CAPTAIN REQUIRED: Key discoveries, root cause, data points, impact assessment]

## Action Items
[CAPTAIN REQUIRED: Concrete recommendations, next steps, ownership]
