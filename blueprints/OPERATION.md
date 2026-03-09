---
# Commander fields (written at dispatch, do not modify)
operation_id: # format: YYYYMMDD-8hex (e.g., 20260212-a1b2c3d4)
squad: # filled by classify (Copilot)
source: user
pid: 0 # written by Commander after launch
created_at: # UTC timestamp
dispatched_at: # UTC timestamp
failed_at: # UTC timestamp
failure_reason: # filled if status is failed
references: [] # filled by classify (Copilot), e.g., ["../20260309-xxxx/"]

# Captain output fields (filled during/after execution)
status: queued # queued → active → completed / failed
summary: # 2-3 sentence summary of outcome
completed_at: # UTC timestamp
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
