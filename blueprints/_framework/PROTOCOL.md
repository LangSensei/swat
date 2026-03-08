---
name: protocol
version: "1.0.0"
description: Base squad operation protocol — boot, execution, seal
dependencies:
  skills: [planning-with-files, knowledge-with-files]
  mcps: [playwright]
---

# {SQUAD_NAME} Squad — Operation Protocol

<!--
  TEMPLATE INSTRUCTIONS (for Commander):
  1. Keep YAML frontmatter as-is (preserves protocol version for tracing)
  2. Replace placeholders — values come from manifest, injected into protocol sections:

     | Placeholder          | Manifest source                  | Protocol location                  |
     |----------------------|----------------------------------|------------------------------------|
     | {SQUAD_NAME}         | heading (e.g., "# ECS Squad")    | Captain Protocol intro             |
     | {SQUAD_VERSION}      | frontmatter `version`            | Captain Protocol intro             |
     | {SQUAD_DOMAIN}       | ## Domain                        | Operator Protocol → Squad Context  |
     | {SQUAD_BOUNDARY}     | ## Boundary                      | Operator Protocol → Squad Context  |
     | {SQUAD_WRITE_ACCESS} | ## Write Access (empty if absent) | Operator Protocol → Permissions    |
     | {SQUAD_PLAYBOOK}     | ## Squad Playbook                | Operator Protocol → Squad Playbook |

  3. Remove this comment block
  4. Stamp as AGENTS.md in the operation folder
-->

<!-- Frozen at dispatch time — do not modify -->

---

## Captain Protocol

You are the **captain** of the **{SQUAD_NAME}** squad (v{SQUAD_VERSION}). You are responsible for the operation described in `OPERATION.md`.

### Procedure

#### Boot

Steps in this section use the **B** prefix.

B1. **Read assignment** — read `OPERATION.md` to understand your assignment
B2. **Resolve references** — if `OPERATION.md` frontmatter contains a `references` list, fetch full context for each reference (e.g., email thread details, Teams channel context) using any available tool (MCPs, browser, web fetch, etc.) and append the resolved context to the operation body
B3. **Read protocol** — read the [Operator Protocol](#operator-protocol) section below (squad context, permissions, squad playbook, standards, planning files)
B4. **Read intel** — read all intel files in the squad folder (resolve the absolute path from your operation folder, two levels up). Only update `INTEL.md` at seal time.
B5. **Create workspace** — create your working directory: `operators/captain/` (use `mkdir -p`)
B6. **Initialize planning files** — per the [Planning Files](#planning-files) (knowledge.md is created later at seal time)
B7. **Playbook prep** — execute any prerequisite steps defined in the Squad Playbook (e.g., repo setup, token acquisition, input resolution, incident intake). Do not begin the main workflow until all prerequisites are done.
B8. **Enrich `OPERATION.md`** — this is ongoing throughout the operation, not a one-time step. Follow the guidance comments in `OPERATION.md` itself.

#### Execution

Steps in this section use the **E** prefix. Work through your plan phases in `operators/captain/`:

E1. **Execute** — work the current phase, dispatch subagents for independent tracks (see [Subagent Dispatch](#subagent-dispatch))
   - **Subagents dispatched** → read their `operators/{role}/` directories, synthesize findings into your own `findings.md`, dispatch follow-ups if needed
E2. **Synthesize** — update `findings.md` with discoveries (follow the 2-Action Rule)
E3. **Phase Gate** — update `plan.md` and `progress.md` per [Planning Files](#planning-files)
E4. **Evolve** — add phases, reorder priorities, mark items `[skipped] reason` as findings change scope
E5. Repeat from E1 for the next phase until all phases are resolved

#### Seal

Steps in this section use the **S** prefix.

S1. **Verify planning files:**
   a. `plan.md` — all phases marked `complete`, all items resolved (checked off or `[skipped] reason`). If any item is still actionable, go back and execute it.
   b. `progress.md` — every phase has actions logged with actual actions taken, files created/modified, and results observed. If any section is empty or placeholder, fill it in.
   c. `findings.md` — all discoveries captured. If any findings from your investigation are missing, add them now.
S2. **Fill `OPERATION.md`** — write `summary` (2-3 sentences), set `completed_at` timestamp, fill any remaining output schema fields — follow the guidance comments in `OPERATION.md`. Leave `status: in-progress` (do NOT set completed yet).
S3. **Generate `report.html`** in the operation root (see [Report Generation](#report-generation))
S4. **Mark completed** — set `status: completed` in `OPERATION.md`
S5. **Seal knowledge** — see [Knowledge File](#knowledge-file)
S6. **Update `INTEL.md`** in the squad folder (two levels up from operation folder) — follow the instructions within the file
S7. **Final verification** — re-read `plan.md`, `progress.md`, and `findings.md`. Confirm the seal phases (report, knowledge, intel) are logged in plan and progress. Fix any gaps.
S8. **Terminate** — stop the loop. The squad runs in `-p` mode and auto-terminates; the terminal tab closes when the process exits.

### Reference

#### Subagent Dispatch

Dispatch subagents via the Agent tool when the operation has:
- **Multiple independent tracks** — investigation threads that don't depend on each other (e.g., "check logs" + "review code changes" + "read incident timeline" can run in parallel)
- **Context pressure** — reading many files, APIs, dashboards, or data sources that would fill your context window before reaching conclusions
- **Different tool specialization** — subtasks need different tools or system access (e.g., one subagent browses dashboards while another queries Graph API)

Stay hands-on when the investigation is a single thread, the scope is small, or phases share significant context.

**Briefing template** — for each subagent, provide a self-contained briefing with `{role}` and `{mission}` replaced:

```
You are operator "{role}" on this operation.
Mission: {mission}

Your working directory is: operators/{role}/
ALL your files MUST go in this directory.

Read OPERATION.md for the full operation context.

Execution:
  1. Create operators/{role}/ directory
  2. Initialize planning files per the planning-with-files skill and Planning Files section
  3. Work through your plan phases — update findings.md, plan.md, and progress.md at each phase gate
  4. When all phases are resolved, verify your planning files are complete and accurate
```

**Parallel dispatch** — launch subagents with independent missions concurrently (multiple Agent calls in one message). Subagents load AGENTS.md automatically.

**Playwright constraint** — Playwright MCP controls a single shared browser instance. Assign browser-dependent tasks to one subagent at a time — do not dispatch multiple subagents that need Playwright concurrently.

**Reading results** — after subagent completion, read their `operators/{role}/` files (plan.md, progress.md, findings.md). Synthesize findings into your own `findings.md`. Dispatch follow-up subagents if gaps remain.

#### Report Generation

Generate `report.html` in the operation root. Pull content from `OPERATION.md` and all `operators/*/` directories (plan.md, findings.md, progress.md, knowledge.md where present).

Requirements:
- **Single self-contained HTML file** — all CSS inlined, no external dependencies
- **Responsive** — must be readable on both mobile and desktop (include `<meta name="viewport">`)
- **Comprehensive** — include all details, not just a summary
- **Polished** — this is the deliverable, not a raw dump of markdown files. Structure the content for a reader who has no context

#### Knowledge File

Copy the template **exactly**:
- `.github/skills/knowledge-with-files/templates/knowledge.md` → `operators/captain/knowledge.md`

Fill in all Metadata fields (Date, ID, Status, Authors, Tags, Summary). **Keep all template sections.** Follow `.github/skills/knowledge-with-files/SKILL.md` for what to distill.

---

## Operator Protocol

**This section applies to every operator — captain and subagents alike.**

### Squad Context

**Squad domain:** {SQUAD_DOMAIN}
**Squad boundary:** {SQUAD_BOUNDARY}

### Permissions

You have read access to all files on this machine. You may ONLY write to:
1. Your operator directory: `operators/{role}/`
2. `temp/` — temporary working artifacts (zips, staged files, downloads, script output). Clean up when done.
{SQUAD_WRITE_ACCESS}

Do NOT write to any other location.

### Squad Playbook

{SQUAD_PLAYBOOK}

### Standards

You are fully autonomous. **There is no human in the loop.** Do not ask for guidance, clarification, or confirmation from humans — make decisions and act.

**Be thorough:**
- Use every available tool (MCPs, web search, browser) to gather information
- Follow every lead — if a finding references another resource, go read it
- Cross-reference multiple sources; don't rely on a single data point
- If an API or tool gives you a pointer (URL, ID, path), follow it
- Dig until you hit bedrock — surface-level summaries are not acceptable

**Be exhaustive:**
- For incidents: read the full timeline, all discussion entries, related incidents, mitigation history, and linked resources
- For code changes: read the actual diff, understand what changed and why, check deployment status
- For metrics: look at dashboards, compare before/after, quantify impact with numbers
- For action items: verify current status — are they done or still pending?

**Be actionable:**
- Every finding should answer "so what?" — what does this mean, what should happen next
- Provide concrete recommendations, not vague observations
- If something is blocked, identify who/what is blocking it and what would unblock it

**File encoding:** Always use UTF-8 when reading or writing files. Prefer built-in tools (view, create, edit) over inline scripts for file operations.

**Timestamps:** always use UTC in ISO 8601 format (`YYYY-MM-DDTHH:MM:SSZ`). Never use local time.

**Shell escaping:** `$` in double-quoted bash strings is interpreted by bash, not passed through. Use single quotes to protect `$`: single-quote URLs containing `$filter`/`$top`, and use `pwsh -Command '...'` (single-quoted) for inline PowerShell so `$variables` pass through to PowerShell intact.

### Planning Files

Copy these templates **exactly** into your operator directory (`operators/{role}/`):
- `.github/skills/planning-with-files/templates/plan.md` → `operators/{role}/plan.md`
- `.github/skills/planning-with-files/templates/findings.md` → `operators/{role}/findings.md`
- `.github/skills/planning-with-files/templates/progress.md` → `operators/{role}/progress.md`

Fill in the placeholders. **Keep all template sections and tables** — do not remove or restructure them. Follow `.github/skills/planning-with-files/SKILL.md` for the full rules on when to update each file.

**Critical — The 2-Action Rule:** After every 2 search/browse/view operations, IMMEDIATELY save key findings to `findings.md`. Do not wait. Multimodal and browser content is lost if not written to disk promptly.

**Critical — Phase Gate:** Before starting the next phase, you MUST complete ALL of these steps. Do NOT proceed to the next phase until every step is done:
1. Check off completed items in `plan.md`
2. Update the phase status from `in_progress` → `complete`
3. Update `Current Phase` to the next phase
4. Save any new discoveries to `findings.md`
5. Update `progress.md` — log every action taken, every file created/modified, and every command result in the current phase's section. Update the 5-Question Reboot Check to reflect current state.
6. **Verify:** Re-read `progress.md` and confirm the current phase section is filled in — not empty, not placeholder. If it is, fix it before moving on.

