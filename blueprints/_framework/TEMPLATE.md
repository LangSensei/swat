---
name: {squad-name}                 # kebab-case identifier, matches folder name
version: "1.0.0"
description: {one-liner description}
dependencies:                      # omit block if no dependencies
  skills: [{skill-names}]
  mcps: [{mcp-names}]
---

# {Squad Name} Squad               <!-- display name, can differ from frontmatter name -->

## Domain
{Detailed scope — what operations, services, or codebases this squad handles.}

## Boundary
- {What the squad DOES — list capabilities}
- Does NOT {what the squad does NOT do — explicit exclusions}

## Write Access
- {paths the squad can write to, or "(none — all interactions via API)" if API-only}

## Squad Playbook
{Domain knowledge, API endpoints, workflows, operational guidance. Use ### subsections to organize content — do not create additional ## sections.}

## Output Schema
Captain must fill these frontmatter fields in `OPERATION.md` during the operation:
```yaml
{field}: # {description}
action_items: [] # Follow-up steps
```
