package commander

import (
	"bufio"
	"fmt"
	"strings"
	"time"
)

// buildOperationFile generates a full OPERATION.md with frontmatter + body template
func buildOperationFile(op *Operation) string {
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString("# Commander fields (written at dispatch, do not modify)\n")
	sb.WriteString("# format: YYYYMMDD-8hex (e.g., 20260212-a1b2c3d4)\n")
	sb.WriteString(fmt.Sprintf("operation_id: %s\n", op.OperationID))
	sb.WriteString("# filled by classify (Copilot)\n")
	sb.WriteString(fmt.Sprintf("squad: %s\n", op.Squad))
	sb.WriteString("# who initiated this operation (user | schedule | system)\n")
	sb.WriteString(fmt.Sprintf("source: %s\n", op.Source))
	sb.WriteString("# written by Commander after launch\n")
	if op.PID > 0 {
		sb.WriteString(fmt.Sprintf("pid: %d\n", op.PID))
	} else {
		sb.WriteString("pid:\n")
	}
	sb.WriteString("# UTC timestamp when operation was created\n")
	sb.WriteString(fmt.Sprintf("created_at: %s\n", op.CreatedAt.Format(time.RFC3339)))
	sb.WriteString("# UTC timestamp when Copilot CLI was launched\n")
	writeOptionalTime(&sb, "dispatched_at", op.DispatchedAt)
	sb.WriteString("# UTC timestamp when operation failed\n")
	writeOptionalTime(&sb, "failed_at", op.FailedAt)
	sb.WriteString("# filled if status is failed\n")
	writeOptionalStr(&sb, "failure_reason", op.FailureReason)
	sb.WriteString("# filled by classify (Copilot)\n")
	sb.WriteString("# e.g., [{type: \"operation\", value: \"../20260309-xxxx/\"}, {type: \"url\", value: \"https://...\"}]\n")
	if len(op.References) > 0 {
		sb.WriteString("references:\n")
		for _, ref := range op.References {
			sb.WriteString(fmt.Sprintf("  - {type: \"%s\", value: \"%s\"}\n", ref.Type, ref.Value))
		}
	} else {
		sb.WriteString("references: []\n")
	}

	sb.WriteString("\n# Captain output fields (filled during/after execution)\n")
	sb.WriteString("# queued → active → completed / failed\n")
	sb.WriteString(fmt.Sprintf("status: %s\n", op.Status))
	sb.WriteString("# 2-3 sentence summary of outcome\n")
	writeOptionalStr(&sb, "summary", nilIfEmpty(op.Summary))
	sb.WriteString("# UTC timestamp when operation completed successfully\n")
	writeOptionalTime(&sb, "completed_at", op.CompletedAt)
	sb.WriteString("---\n\n")

	// Body
	briefTitle := op.Brief
	if runeCount := len([]rune(briefTitle)); runeCount > 60 {
		runes := []rune(briefTitle)
		briefTitle = string(runes[:60]) + "..."
	}
	if idx := strings.Index(briefTitle, "\n"); idx > 0 {
		briefTitle = briefTitle[:idx]
	}
	sb.WriteString(fmt.Sprintf("# %s\n", briefTitle))
	sb.WriteString("<!-- Commander: extracted from brief — do not modify -->\n\n")

	sb.WriteString("## Assignment\n")
	sb.WriteString("<!-- Commander: full operation description — do not modify -->\n")
	sb.WriteString(op.Brief + "\n")
	if op.Details != "" {
		sb.WriteString("\n" + op.Details + "\n")
	}

	sb.WriteString("\n## Summary\n")
	sb.WriteString("<!-- Captain: write a rich summary of findings and outcome -->\n\n")

	sb.WriteString("## Findings\n")
	sb.WriteString("<!-- Captain: key discoveries, root cause, impact, affected environments -->\n\n")

	sb.WriteString("## Action Items\n")
	sb.WriteString("<!-- Captain: concrete recommendations and next steps -->\n")

	return sb.String()
}

func writeOptionalTime(sb *strings.Builder, key string, t *time.Time) {
	if t != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", key, t.Format(time.RFC3339)))
	} else {
		sb.WriteString(fmt.Sprintf("%s:\n", key))
	}
}

func writeOptionalStr(sb *strings.Builder, key string, s *string) {
	if s != nil && *s != "" {
		sb.WriteString(fmt.Sprintf("%s: %s\n", key, *s))
	} else {
		sb.WriteString(fmt.Sprintf("%s:\n", key))
	}
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// parseOperationMD parses an OPERATION.md file into an Operation struct
func parseOperationMD(content string) (*Operation, error) {
	if !strings.HasPrefix(content, "---") {
		return nil, fmt.Errorf("missing frontmatter")
	}
	end := strings.Index(content[3:], "\n---")
	if end < 0 {
		return nil, fmt.Errorf("unterminated frontmatter")
	}
	fm := content[4 : end+3]
	body := content[end+7:] // after closing ---

	op := &Operation{}
	scanner := bufio.NewScanner(strings.NewReader(fm))
	for scanner.Scan() {
		line := scanner.Text()
		// Skip YAML comments
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		switch key {
		case "operation_id":
			op.OperationID = val
		case "squad":
			op.Squad = val
		case "status":
			op.Status = val
		case "source":
			op.Source = val
		case "pid":
			fmt.Sscanf(val, "%d", &op.PID)
		case "created_at":
			if t, err := time.Parse(time.RFC3339, val); err == nil {
				op.CreatedAt = t
			}
		case "dispatched_at":
			if t, err := time.Parse(time.RFC3339, val); err == nil {
				op.DispatchedAt = &t
			}
		case "completed_at":
			if t, err := time.Parse(time.RFC3339, val); err == nil {
				op.CompletedAt = &t
			}
		case "failed_at":
			if t, err := time.Parse(time.RFC3339, val); err == nil {
				op.FailedAt = &t
			}
		case "failure_reason":
			if val != "" {
				op.FailureReason = &val
			}
		case "summary":
			op.Summary = val
		}
	}

	// Parse brief from body: content under ## Assignment
	op.Brief = extractBodySection(body, "Assignment")

	if op.OperationID == "" {
		return nil, fmt.Errorf("missing operation_id in frontmatter")
	}
	return op, nil
}

// extractBodySection extracts the content under a ## heading
func extractBodySection(body, heading string) string {
	marker := "## " + heading
	idx := strings.Index(body, marker)
	if idx < 0 {
		return ""
	}
	rest := body[idx+len(marker):]
	// Skip the heading line
	if nl := strings.Index(rest, "\n"); nl >= 0 {
		rest = rest[nl+1:]
	}
	// Skip HTML comments
	for strings.HasPrefix(strings.TrimSpace(rest), "<!--") {
		if end := strings.Index(rest, "-->"); end >= 0 {
			rest = rest[end+3:]
			rest = strings.TrimLeft(rest, "\n")
		} else {
			break
		}
	}
	// Read until next ## or end
	if next := strings.Index(rest, "\n## "); next >= 0 {
		rest = rest[:next]
	}
	return strings.TrimSpace(rest)
}
