package commander

import (
	"encoding/json"
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

// composeMCPConfig builds .mcp.json from individual MCP config files.
// runtimeName and notifyBackend are injected as --runtime and --notify flags
// into the "swat" server args, if present.
func composeMCPConfig(swatRoot, runtimeName, notifyBackend string, mcps []string) string {
	mcpsDir := filepath.Join(swatRoot, "blueprints", "mcps")
	servers := make(map[string]string)
	for _, name := range mcps {
		path := filepath.Join(mcpsDir, name+".json")
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		raw := strings.TrimSpace(string(data))

		// Inject --runtime and --notify into the swat server entry
		if name == "swat" {
			raw = injectSwatArgs(raw, runtimeName, notifyBackend)
		}
		servers[name] = raw
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

// injectSwatArgs adds --runtime and --notify flags to a swat MCP server JSON config.
func injectSwatArgs(raw, runtimeName, notifyBackend string) string {
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &obj); err != nil {
		return raw
	}

	var existing []string
	if args, ok := obj["args"]; ok {
		if arr, ok := args.([]interface{}); ok {
			for _, a := range arr {
				if s, ok := a.(string); ok {
					existing = append(existing, s)
				}
			}
		}
	}

	if runtimeName != "" {
		existing = append(existing, "--runtime", runtimeName)
	}
	if notifyBackend != "" {
		existing = append(existing, "--notify", notifyBackend)
	}
	obj["args"] = existing

	out, err := json.Marshal(obj)
	if err != nil {
		return raw
	}
	return string(out)
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
