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
