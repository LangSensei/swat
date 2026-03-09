---
# Commander fields (written at dispatch, do not modify)
# format: YYYYMMDD-8hex (e.g., 20260212-a1b2c3d4)
operation_id:
# filled by classify (Copilot)
squad:
# who initiated this operation
source: user
# written by Commander after launch
pid: 0
# UTC timestamp when operation was created
created_at:
# UTC timestamp when Copilot CLI was launched
dispatched_at:
# UTC timestamp when operation failed
failed_at:
# filled if status is failed
failure_reason:
# filled by classify (Copilot)
# e.g., [{type: "operation", value: "../20260309-xxxx/"}, {type: "url", value: "https://..."}, {type: "email-address", value: "user@example.com"}]
references: []

# Captain output fields (filled during/after execution)
# queued → active → completed / failed
status: queued
# 2-3 sentence summary of outcome
summary:
# UTC timestamp when operation completed successfully
completed_at:
---

# {BRIEF_TITLE}
<!-- Commander: extracted from brief — do not modify -->

## Assignment
<!-- Commander: full operation description — do not modify -->
{DESCRIPTION}

## Summary
<!-- Captain: write a rich summary of findings and outcome -->

## Findings
<!-- Captain: key discoveries, root cause, impact, affected environments -->

## Action Items
<!-- Captain: concrete recommendations and next steps -->
