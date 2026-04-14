package deps

import "strings"

// ParseDependencyList extracts a dependency list from frontmatter, e.g. "skills: [a, b]".
func ParseDependencyList(md, field string) []string {
	if !strings.HasPrefix(md, "---") {
		return nil
	}
	end := strings.Index(md[3:], "\n---")
	if end < 0 {
		return nil
	}
	fm := md[4 : end+3]
	for _, line := range strings.Split(fm, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, field+":") {
			val := strings.TrimSpace(strings.TrimPrefix(trimmed, field+":"))
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
	}
	return nil
}

// ExtractFrontmatterField extracts a single field value from YAML frontmatter.
func ExtractFrontmatterField(md, field string) string {
	if !strings.HasPrefix(md, "---") {
		return ""
	}
	end := strings.Index(md[3:], "\n---")
	if end < 0 {
		return ""
	}
	fm := md[4 : end+3]
	for _, line := range strings.Split(fm, "\n") {
		if strings.HasPrefix(line, field+":") {
			val := strings.TrimSpace(strings.TrimPrefix(line, field+":"))
			val = strings.Trim(val, "\"")
			return val
		}
	}
	return ""
}

// ParseFrontmatterValue extracts a single string value from frontmatter (trims quotes).
func ParseFrontmatterValue(md, field string) string {
	if !strings.HasPrefix(md, "---") {
		return ""
	}
	end := strings.Index(md[3:], "\n---")
	if end < 0 {
		return ""
	}
	fm := md[4 : end+3]
	for _, line := range strings.Split(fm, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, field+":") {
			val := strings.TrimSpace(strings.TrimPrefix(trimmed, field+":"))
			val = strings.Trim(val, "\"'")
			return val
		}
	}
	return ""
}
