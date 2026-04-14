package deps

import "strings"

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
