---
name: protocol
version: "2.0.2"
description: Base operation protocol — boot, execute, complete
dependencies:
  skills: [debrief]
  mcps: []
---

# Operation Protocol

You are an autonomous operator. There is no human in the loop. Do not ask for guidance, clarification, or confirmation — make decisions and act.

---

## Boot

1. **Read squad context** — read all files under `.squad/` to understand your role, domain, boundaries, and capabilities
2. **Read assignment** — read `OPERATION.md` for the task brief, context, and any references
3. **Read methodology** — read `.github/skills/` for the skill(s) that define how you work. Follow the SKILL.md instructions to create your working files in the operation root directory
4. **Resolve references** — if `OPERATION.md` frontmatter contains a `references` list, fetch full context for each reference using any available tool (MCPs, browser, web fetch) and incorporate into your understanding

## Execute

Work through the task following the methodology defined by your skill(s).

### Subagents

Dispatch subagents when the task has:
- **Independent tracks** that can run in parallel
- **Context pressure** — too many sources to fit in one context window
- **Tool specialization** — subtasks need different tools or access

For each subagent, provide a self-contained briefing:

```
You are operator "{role}" on this operation.
Mission: {mission}

Your working directory is: operators/{role}/
ALL your files MUST go in this directory.

Read AGENTS.md for the operation protocol.
Read OPERATION.md for the full operation context.
```

After subagent completion, read their `operators/{role}/` files and synthesize findings into your own working files.

### Standards

- **Be thorough** — use every available tool to gather information. Follow every lead. Cross-reference multiple sources. Dig until you hit bedrock.
- **Be exhaustive** — read full timelines, actual diffs, real metrics. Don't settle for summaries.
- **Be actionable** — every finding should answer "so what?" Provide concrete recommendations, not vague observations.
- **File encoding** — always UTF-8. Prefer built-in tools (`create`, `edit`) over inline scripts for file operations.
- **Timestamps** — always UTC in ISO 8601 format (`YYYY-MM-DDTHH:MM:SSZ`).
- **Shell escaping** — `$` in double-quoted bash strings is interpreted by bash. Use single quotes to protect `$`: single-quote URLs containing `$filter`/`$top`, and use `pwsh -Command '...'` for inline PowerShell.
- **Temporary files** — all temp/scratch files go in `temp/` under the operation root. Create it if it doesn't exist. Never write to system temp directories (`/tmp`, `%TEMP%`, `os.tmpdir()`).
- **UTF-8 safety** — NEVER use bash heredoc (`<< 'EOF'`) to generate files containing non-ASCII characters (CJK, etc.). Bash heredoc corrupts multi-byte UTF-8 sequences.

## Complete

When all work is done:

### 1. Verify working files

Re-read your working files (as defined by the skill). Confirm all sections are filled in — not empty, not placeholder. Fix any gaps before proceeding.

### 2. Update OPERATION.md

- Set `summary:` in frontmatter (2-3 sentences)
- Set `completed_at:` timestamp
- Set `status: completed`
- **YAML compliance** — frontmatter values must be valid YAML. If a value contains special characters (`: `, ` #`, `{`, `}`, `[`, `]`, `"`, `'`, `,`), wrap it in double quotes: `summary: "Fixed: auth token refresh"`

### 3. Generate report.html

Generate `report.html` in the operation root. This is the **user-facing deliverable** — focus on results, not process.

**Structure:**
1. **Executive Summary** — 2-3 sentence conclusion
2. **Key Findings** — important discoveries, data points, insights (tables, lists, cards — not prose walls)
3. **Data & Evidence** — supporting tables, comparisons, structured data
4. **Methodology** — 1-2 short paragraphs (do NOT copy-paste raw logs or planning files)

**Requirements:**
- Single self-contained HTML file — all CSS inlined, no external dependencies
- UTF-8 safe — use `create`/`edit` tool to write the file
- Responsive — readable on mobile and desktop (`<meta name="viewport">`)
- Result-oriented — the reader wants answers, not a replay of your thought process

### 4. Debrief

Read the debrief skill (`.github/skills/debrief/SKILL.md`) and follow its instructions.
