package deps

import (
	"fmt"
	"os"
	"strings"
)

// parseFrontmatter extracts YAML frontmatter key-value pairs from markdown.
// Returns nil if no valid frontmatter block is found.
func parseFrontmatter(md string) map[string]string {
	if !strings.HasPrefix(md, "---") {
		return nil
	}
	end := strings.Index(md[3:], "\n---")
	if end < 0 {
		return nil
	}
	fm := md[4 : end+3]
	result := make(map[string]string)
	for _, line := range strings.Split(fm, "\n") {
		trimmed := strings.TrimSpace(line)
		idx := strings.Index(trimmed, ":")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(trimmed[:idx])
		val := strings.TrimSpace(trimmed[idx+1:])
		if key != "" {
			result[key] = val
		}
	}
	return result
}

// ParseDependencyList extracts a dependency list from frontmatter, e.g. "skills: [a, b]".
func ParseDependencyList(md, field string) []string {
	fm := parseFrontmatter(md)
	if fm == nil {
		return nil
	}
	val, ok := fm[field]
	if !ok {
		return nil
	}
	val = strings.Trim(val, "[]")
	if val == "" {
		return nil
	}
	var result []string
	for _, d := range strings.Split(val, ",") {
		d = strings.TrimSpace(d)
		if d != "" {
			result = append(result, d)
		}
	}
	return result
}

// ExtractFrontmatterField extracts a single field value from YAML frontmatter.
// Strips surrounding single and double quotes.
func ExtractFrontmatterField(md, field string) string {
	fm := parseFrontmatter(md)
	if fm == nil {
		return ""
	}
	val, ok := fm[field]
	if !ok {
		return ""
	}
	return strings.Trim(val, "\"'")
}

// PatchFrontmatterFields updates specific key-value pairs in a YAML frontmatter file.
func PatchFrontmatterFields(path string, patches map[string]string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	content := string(data)
	if !strings.HasPrefix(content, "---") {
		return fmt.Errorf("missing frontmatter")
	}

	end := strings.Index(content[3:], "\n---")
	if end < 0 {
		return fmt.Errorf("unterminated frontmatter")
	}

	fm := content[4 : end+3]
	body := content[end+7:]

	lines := strings.Split(fm, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") || trimmed == "" {
			continue
		}
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		if val, ok := patches[key]; ok {
			lines[i] = key + ": " + val
		}
	}

	result := "---\n" + strings.Join(lines, "\n") + "\n---" + body
	return os.WriteFile(path, []byte(result), 0644)
}
