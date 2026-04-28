package operation

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/LangSensei/swat/commander/layout"
	"gopkg.in/yaml.v3"
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

	// Commander fields
	result = strings.ReplaceAll(result, "{OPERATION_ID}", op.OperationID)
	result = strings.ReplaceAll(result, "{SQUAD}", op.Squad)
	if op.PID > 0 {
		result = strings.ReplaceAll(result, "{PID}", fmt.Sprintf("%d", op.PID))
	} else {
		result = strings.ReplaceAll(result, "{PID}", "")
	}

	result = strings.ReplaceAll(result, "{CREATED_AT}", op.CreatedAt.Format(time.RFC3339))
	result = strings.ReplaceAll(result, "{DISPATCHED_AT}", formatOptionalTime(op.DispatchedAt))
	result = strings.ReplaceAll(result, "{FAILED_AT}", formatOptionalTime(op.FailedAt))
	result = strings.ReplaceAll(result, "{FAILURE_REASON}", formatOptionalStr(op.FailureReason))

	// References require special handling for YAML list format
	if len(op.References) > 0 {
		var refStr strings.Builder
		refStr.WriteString("references:")
		for _, ref := range op.References {
			refStr.WriteString(fmt.Sprintf("\n  - {type: \"%s\", value: \"%s\"}", ref.Type, ref.Value))
		}
		result = strings.Replace(result, "references: {REFERENCES}", refStr.String(), 1)
	} else {
		result = strings.ReplaceAll(result, "{REFERENCES}", "[]")
	}

	// Captain output fields
	result = strings.ReplaceAll(result, "{STATUS}", op.Status)
	result = strings.ReplaceAll(result, "{SUMMARY}", op.Summary)
	result = strings.ReplaceAll(result, "{COMPLETED_AT}", formatOptionalTime(op.CompletedAt))

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

func formatOptionalTime(t *time.Time) string {
	if t != nil {
		return t.Format(time.RFC3339)
	}
	return ""
}

func formatOptionalStr(s *string) string {
	if s != nil && *s != "" {
		return *s
	}
	return ""
}

// parseOperationMD parses an OPERATION.md file into an Operation struct.
// Frontmatter is parsed with yaml.v3; body (Brief/Details) is extracted with string logic.
func parseOperationMD(content string) (*Operation, error) {
	fmStr, body, err := splitFrontmatter(content)
	if err != nil {
		return nil, err
	}

	op := &Operation{}
	if err := yaml.Unmarshal([]byte(fmStr), op); err != nil {
		return nil, fmt.Errorf("unmarshal frontmatter: %w", err)
	}

	body = strings.TrimLeft(body, "\n")

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

// splitFrontmatter extracts the raw YAML frontmatter string and body from markdown content.
func splitFrontmatter(content string) (string, string, error) {
	if !strings.HasPrefix(content, "---") {
		return "", "", fmt.Errorf("missing frontmatter")
	}
	end := strings.Index(content[3:], "\n---")
	if end < 0 {
		return "", "", fmt.Errorf("unterminated frontmatter")
	}
	fm := content[4 : end+3]
	body := content[end+7:]
	return fm, body, nil
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
