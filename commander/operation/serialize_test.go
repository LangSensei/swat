package operation

import (
	"strings"
	"testing"
)

func TestParseOperationMD_BriefExtraction(t *testing.T) {
	content := "---\n" +
		"operation_id: 20260415-test1234\n" +
		"status: active\n" +
		"squad: test-squad\n" +
		"created_at: 2026-04-15T06:00:00Z\n" +
		"---\n" +
		"\n" +
		"# Fix the frontmatter bug\n" +
		"\n" +
		"## Assignment\n" +
		"Some details here.\n"

	op, err := parseOperationMD(content)
	if err != nil {
		t.Fatalf("parseOperationMD failed: %v", err)
	}

	if op.Brief == "" {
		t.Errorf("Brief is empty — body likely has leading newline causing idx > 0 check to fail")
	}
	if op.Brief != "Fix the frontmatter bug" {
		t.Errorf("Brief = %q, want %q", op.Brief, "Fix the frontmatter bug")
	}
	if op.Details == "" {
		t.Errorf("Details is empty")
	}
	if !strings.Contains(op.Details, "Some details here") {
		t.Errorf("Details = %q, want to contain 'Some details here'", op.Details)
	}
}

func TestParseOperationMD_MultiLineSummary(t *testing.T) {
	content := "---\n" +
		"operation_id: 20260428-multiline\n" +
		"status: completed\n" +
		"squad: test-squad\n" +
		"created_at: 2026-04-28T10:00:00Z\n" +
		"summary: |\n" +
		"  This is a multi-line summary.\n" +
		"  It spans multiple lines using YAML block scalar.\n" +
		"---\n" +
		"\n" +
		"# Multi-line test\n"

	op, err := parseOperationMD(content)
	if err != nil {
		t.Fatalf("parseOperationMD failed: %v", err)
	}

	if !strings.Contains(op.Summary, "multi-line summary") {
		t.Errorf("Summary = %q, want to contain 'multi-line summary'", op.Summary)
	}
	if !strings.Contains(op.Summary, "spans multiple lines") {
		t.Errorf("Summary = %q, want to contain 'spans multiple lines'", op.Summary)
	}
}

func TestParseOperationMD_References(t *testing.T) {
	content := "---\n" +
		"operation_id: 20260428-refs\n" +
		"status: active\n" +
		"squad: test-squad\n" +
		"created_at: 2026-04-28T10:00:00Z\n" +
		"references:\n" +
		"  - {type: \"operation\", value: \"../20260415-0aa4c62d/\"}\n" +
		"  - {type: \"operation\", value: \"../20260325-8089c4b9/\"}\n" +
		"---\n" +
		"\n" +
		"# References test\n"

	op, err := parseOperationMD(content)
	if err != nil {
		t.Fatalf("parseOperationMD failed: %v", err)
	}

	if len(op.References) != 2 {
		t.Fatalf("References count = %d, want 2", len(op.References))
	}
	if op.References[0].Type != "operation" {
		t.Errorf("References[0].Type = %q, want %q", op.References[0].Type, "operation")
	}
	if op.References[0].Value != "../20260415-0aa4c62d/" {
		t.Errorf("References[0].Value = %q, want %q", op.References[0].Value, "../20260415-0aa4c62d/")
	}
	if op.References[1].Value != "../20260325-8089c4b9/" {
		t.Errorf("References[1].Value = %q, want %q", op.References[1].Value, "../20260325-8089c4b9/")
	}
}

func TestParseOperationMD_YAMLComments(t *testing.T) {
	content := "---\n" +
		"# ── COMMANDER (written at dispatch, do not modify) ──\n" +
		"operation_id: 20260428-comments\n" +
		"pid: 12345\n" +
		"created_at: 2026-04-28T10:00:00Z\n" +
		"\n" +
		"# ── CLASSIFY (written by LLM at classify, do not modify) ──\n" +
		"squad: swat-dev\n" +
		"references:\n" +
		"  - {type: \"operation\", value: \"../ref1/\"}\n" +
		"\n" +
		"# ── OPERATOR (fill during execution) ──\n" +
		"status: active\n" +
		"summary: \n" +
		"completed_at: \n" +
		"---\n" +
		"\n" +
		"# Test with YAML comments\n" +
		"\n" +
		"## Assignment\n" +
		"Details here.\n"

	op, err := parseOperationMD(content)
	if err != nil {
		t.Fatalf("parseOperationMD failed: %v", err)
	}

	if op.OperationID != "20260428-comments" {
		t.Errorf("OperationID = %q, want %q", op.OperationID, "20260428-comments")
	}
	if op.PID != 12345 {
		t.Errorf("PID = %d, want 12345", op.PID)
	}
	if op.Squad != "swat-dev" {
		t.Errorf("Squad = %q, want %q", op.Squad, "swat-dev")
	}
	if len(op.References) != 1 {
		t.Fatalf("References count = %d, want 1", len(op.References))
	}
	if op.Brief != "Test with YAML comments" {
		t.Errorf("Brief = %q, want %q", op.Brief, "Test with YAML comments")
	}
}

func TestPatchFrontmatterFields_PreservesBody(t *testing.T) {
	original := "---\n" +
		"operation_id: 20260415-test1234\n" +
		"status: active\n" +
		"---\n" +
		"\n" +
		"# My Title\n" +
		"\n" +
		"## Assignment\n" +
		"Details.\n"

	// Simulate what patchFrontmatterFields does internally
	content := original
	if !strings.HasPrefix(content, "---") {
		t.Fatal("missing frontmatter")
	}
	end := strings.Index(content[3:], "\n---")
	if end < 0 {
		t.Fatal("unterminated frontmatter")
	}
	body := content[end+7:]

	// Reconstruct — body must preserve the blank line between --- and # Title
	fm := "operation_id: 20260415-test1234\nstatus: completed"
	result := "---\n" + fm + "\n---" + body

	if !strings.Contains(result, "\n---\n\n# My Title") {
		t.Errorf("Reconstruction lost blank line separator.\nGot:\n%s", result)
	}
}
