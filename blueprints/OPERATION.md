---
# Commander fields (written at dispatch, do not modify)
# format: YYYYMMDD-8hex (e.g., 20260212-a1b2c3d4)
operation_id: {OPERATION_ID}
# filled by classify (Copilot)
squad: {SQUAD}
# who initiated this operation (user | schedule | system)
source: {SOURCE}
# written by Commander after launch
pid: {PID}
# UTC timestamp when operation was created
created_at: {CREATED_AT}
# UTC timestamp when Copilot CLI was launched
dispatched_at: {DISPATCHED_AT}
# UTC timestamp when operation failed
failed_at: {FAILED_AT}
# filled if status is failed
failure_reason: {FAILURE_REASON}
# filled by classify (Copilot)
# e.g., [{type: "operation", value: "../20260309-xxxx/"}, {type: "url", value: "https://..."}, {type: "email-address", value: "user@example.com"}]
references: {REFERENCES}

# Captain output fields (filled during/after execution)
# queued → active → completed / failed
status: {STATUS}
# 2-3 sentence summary of outcome
summary: {SUMMARY}
# UTC timestamp when operation completed successfully
completed_at: {COMPLETED_AT}
# Squad-specific output fields
# {OUTPUT_SCHEMA}
---

# {BRIEF}
<!-- Commander: extracted from brief — do not modify -->

## Assignment
<!-- Commander: full operation description — do not modify -->
{DETAILS}

## Summary
<!-- Captain: write a rich summary of findings and outcome -->

## Findings
<!-- Captain: key discoveries, root cause, impact, affected environments -->

## Action Items
<!-- Captain: concrete recommendations and next steps -->
