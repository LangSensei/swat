package operation

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/LangSensei/swat/commander/deps"
	"github.com/LangSensei/swat/commander/layout"
)

// buildOperationFile reads blueprints/OPERATION.md as a template and replaces
// placeholders with values from the Operation struct.
func buildOperationFile(op *Operation) (string, error) {
	tmplPath := layout.OperationTemplatePath()
	tmplData, err := os.ReadFile(tmplPath)
	if err != nil {
		return "", fmt.Errorf("read operation template: %w", err)
	}

	result := string(tmplData)

	// Dynamic fields filled at creation time
	result = strings.ReplaceAll(result, "{OPERATION_ID}", op.OperationID)
	result = strings.ReplaceAll(result, "{CREATED_AT}", op.CreatedAt.Format(time.RFC3339))

	// Body placeholders
	briefTitle := op.Brief
	if idx := strings.Index(briefTitle, "\n"); idx > 0 {
		briefTitle = briefTitle[:idx]
	}
	result = strings.ReplaceAll(result, "{BRIEF}", briefTitle)

	if op.Details != "" {
		result = strings.ReplaceAll(result, "{DETAILS}", op.Details)
	} else {
		result = strings.ReplaceAll(result, "{DETAILS}", "")
	}

	return result, nil
}

// parseOperationMD parses an OPERATION.md file into an Operation struct.
func parseOperationMD(content string) (*Operation, error) {
	fm, body, err := deps.ParseFrontmatter(content)
	if err != nil {
		return nil, err
	}
	body = strings.TrimLeft(body, "\n") // skip newlines after closing ---

	op := &Operation{}
	for key, val := range fm {
		switch key {
		case "operation_id":
			op.OperationID = val
		case "squad":
			op.Squad = val
		case "status":
			op.Status = val
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

	// Parse brief from body: first H1 title
	if idx := strings.Index(body, "\n"); idx > 0 {
		title := strings.TrimSpace(body[:idx])
		title = strings.TrimPrefix(title, "# ")
		op.Brief = title
	}
	// Parse details from body: content under ## Assignment
	op.Details = extractBodySection(body, "Assignment")

	if op.OperationID == "" {
		return nil, fmt.Errorf("missing operation_id in frontmatter")
	}
	return op, nil
}

// extractBodySection extracts the content under a ## heading.
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
