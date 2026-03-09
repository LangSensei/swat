package commander

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// resolveDependencies recursively resolves all skill dependencies for a squad
func (c *Commander) resolveDependencies(squad string) []string {
	bpDir := filepath.Join(c.SwatRoot, "blueprints")
	visited := make(map[string]bool)
	var result []string

	// Collect initial skills from protocol and manifest
	var seeds []string
	protocolPath := filepath.Join(bpDir, "squads", "_framework", "PROTOCOL.md")
	if data, err := os.ReadFile(protocolPath); err == nil {
		seeds = append(seeds, parseDependencyList(string(data), "skills")...)
	}
	manifestPath := filepath.Join(bpDir, "squads", squad, "MANIFEST.md")
	if data, err := os.ReadFile(manifestPath); err == nil {
		seeds = append(seeds, parseDependencyList(string(data), "skills")...)
	}

	// BFS to resolve transitive dependencies
	queue := seeds
	for len(queue) > 0 {
		skill := queue[0]
		queue = queue[1:]
		if visited[skill] {
			continue
		}
		visited[skill] = true
		result = append(result, skill)

		// Check skill's own dependencies
		skillMD := filepath.Join(c.SwatRoot, "blueprints", "skills", skill, "SKILL.md")
		if data, err := os.ReadFile(skillMD); err == nil {
			for _, dep := range parseDependencyList(string(data), "skills") {
				if !visited[dep] {
					queue = append(queue, dep)
				}
			}
		}
	}
	return result
}

// resolveMCPDependencies collects all MCP names from protocol, manifest, and transitive skills
func (c *Commander) resolveMCPDependencies(squad string) []string {
	bpDir := filepath.Join(c.SwatRoot, "blueprints")
	seen := make(map[string]bool)
	var result []string

	// Protocol MCPs
	if data, err := os.ReadFile(filepath.Join(bpDir, "squads", "_framework", "PROTOCOL.md")); err == nil {
		for _, m := range parseDependencyList(string(data), "mcps") {
			if !seen[m] {
				seen[m] = true
				result = append(result, m)
			}
		}
	}
	// Manifest MCPs
	if data, err := os.ReadFile(filepath.Join(bpDir, "squads", squad, "MANIFEST.md")); err == nil {
		for _, m := range parseDependencyList(string(data), "mcps") {
			if !seen[m] {
				seen[m] = true
				result = append(result, m)
			}
		}
	}
	// Transitive MCPs from resolved skills
	for _, skill := range c.resolveDependencies(squad) {
		skillMD := filepath.Join(c.SwatRoot, "blueprints", "skills", skill, "SKILL.md")
		if data, err := os.ReadFile(skillMD); err == nil {
			for _, m := range parseDependencyList(string(data), "mcps") {
				if !seen[m] {
					seen[m] = true
					result = append(result, m)
				}
			}
		}
	}
	return result
}

// composeMCPConfig builds .mcp.json from individual MCP config files
func composeMCPConfig(swatRoot string, mcps []string) string {
	mcpsDir := filepath.Join(swatRoot, "blueprints", "mcps")
	servers := make(map[string]string)
	for _, name := range mcps {
		path := filepath.Join(mcpsDir, name+".json")
		if data, err := os.ReadFile(path); err == nil {
			servers[name] = strings.TrimSpace(string(data))
		}
	}
	if len(servers) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("{\n  \"mcpServers\": {\n")
	i := 0
	for name, config := range servers {
		if i > 0 {
			sb.WriteString(",\n")
		}
		sb.WriteString(fmt.Sprintf("    %q: %s", name, config))
		i++
	}
	sb.WriteString("\n  }\n}\n")
	return sb.String()
}

// parseDependencyList extracts a dependency list from frontmatter, e.g. "skills: [a, b]"
func parseDependencyList(md, field string) []string {
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
			var deps []string
			for _, d := range strings.Split(val, ",") {
				d = strings.TrimSpace(d)
				if d != "" {
					deps = append(deps, d)
				}
			}
			return deps
		}
	}
	return nil
}

// assembleAgentsMD replaces placeholders in PROTOCOL.md with manifest sections
func assembleAgentsMD(manifest, protocol, squadName string) string {
	domain := extractSection(manifest, "## Domain")
	boundary := extractSection(manifest, "## Boundary")
	writeAccess := extractSection(manifest, "## Write Access")
	playbook := extractSection(manifest, "## Squad Playbook")
	version := extractFrontmatterField(manifest, "version")
	if version == "" {
		version = "1.0.0"
	}

	result := stripFrontmatter(protocol)
	result = strings.ReplaceAll(result, "{SQUAD_NAME}", squadName)
	result = strings.ReplaceAll(result, "{SQUAD_VERSION}", version)
	result = strings.ReplaceAll(result, "{SQUAD_DOMAIN}", domain)
	result = strings.ReplaceAll(result, "{SQUAD_BOUNDARY}", boundary)
	result = strings.ReplaceAll(result, "{SQUAD_WRITE_ACCESS}", writeAccess)
	result = strings.ReplaceAll(result, "{SQUAD_PLAYBOOK}", playbook)
	return result
}

// extractSection extracts content under a markdown heading
func extractSection(md, heading string) string {
	idx := strings.Index(md, heading)
	if idx < 0 {
		return ""
	}
	content := md[idx+len(heading):]
	if nextIdx := strings.Index(content, "\n## "); nextIdx >= 0 {
		content = content[:nextIdx]
	}
	return strings.TrimSpace(content)
}

// extractFrontmatterField extracts a single field value from YAML frontmatter
func extractFrontmatterField(md, field string) string {
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

// stripFrontmatter removes YAML frontmatter from a markdown document
func stripFrontmatter(md string) string {
	if !strings.HasPrefix(md, "---") {
		return md
	}
	end := strings.Index(md[3:], "---")
	if end < 0 {
		return md
	}
	return strings.TrimLeft(md[end+6:], "\n")
}

// extractOutputSchema extracts YAML fields from the ## Output Schema section of a MANIFEST.md
// It looks for the yaml code block and returns the field lines (without the ```yaml wrapper)
func extractOutputSchema(manifest string) string {
	body := stripFrontmatter(manifest)

	// Find ## Output Schema section
	idx := strings.Index(body, "## Output Schema")
	if idx < 0 {
		return ""
	}
	section := body[idx:]

	// Find yaml code block
	codeStart := strings.Index(section, "```yaml")
	if codeStart < 0 {
		return ""
	}
	afterOpen := section[codeStart+7:]

	codeEnd := strings.Index(afterOpen, "```")
	if codeEnd < 0 {
		return ""
	}

	return strings.TrimSpace(afterOpen[:codeEnd])
}

// injectOutputSchema reads the squad MANIFEST, extracts Output Schema fields,
// and replaces the {SQUAD_OUTPUT_SCHEMA} placeholder in OPERATION.md
func (c *Commander) injectOutputSchema(squad string, opDir string) error {
	manifestPath := filepath.Join(c.SwatRoot, "blueprints", "squads", squad, "MANIFEST.md")
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}

	schema := extractOutputSchema(string(manifest))

	opMDPath := filepath.Join(opDir, "OPERATION.md")
	content, err := os.ReadFile(opMDPath)
	if err != nil {
		return fmt.Errorf("read OPERATION.md: %w", err)
	}

	placeholder := "{OUTPUT_SCHEMA}"
	contentStr := string(content)
	if !strings.Contains(contentStr, placeholder) {
		return nil
	}

	replaced := strings.Replace(contentStr, placeholder, schema, 1)
	return os.WriteFile(opMDPath, []byte(replaced), 0644)
}
