package deps

import (
	"os"
	"path/filepath"
)

// ResolveDependencies recursively resolves all skill dependencies for a squad.
func ResolveDependencies(swatRoot, squad string) []string {
	bpDir := filepath.Join(swatRoot, "blueprints")
	visited := make(map[string]bool)
	var result []string

	// Collect initial skills from protocol and manifest
	var seeds []string
	protocolPath := filepath.Join(bpDir, "squads", "_framework", "PROTOCOL.md")
	if data, err := os.ReadFile(protocolPath); err == nil {
		seeds = append(seeds, ParseDependencyList(string(data), "skills")...)
	}
	manifestPath := filepath.Join(bpDir, "squads", squad, "MANIFEST.md")
	if data, err := os.ReadFile(manifestPath); err == nil {
		seeds = append(seeds, ParseDependencyList(string(data), "skills")...)
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
		skillMD := filepath.Join(swatRoot, "blueprints", "skills", skill, "SKILL.md")
		if data, err := os.ReadFile(skillMD); err == nil {
			for _, dep := range ParseDependencyList(string(data), "skills") {
				if !visited[dep] {
					queue = append(queue, dep)
				}
			}
		}
	}
	return result
}

// ResolveMCPDependencies collects all MCP names from protocol, manifest, and transitive skills.
func ResolveMCPDependencies(swatRoot, squad string) []string {
	bpDir := filepath.Join(swatRoot, "blueprints")
	seen := make(map[string]bool)
	var result []string

	// Protocol MCPs
	if data, err := os.ReadFile(filepath.Join(bpDir, "squads", "_framework", "PROTOCOL.md")); err == nil {
		for _, m := range ParseDependencyList(string(data), "mcps") {
			if !seen[m] {
				seen[m] = true
				result = append(result, m)
			}
		}
	}
	// Manifest MCPs
	if data, err := os.ReadFile(filepath.Join(bpDir, "squads", squad, "MANIFEST.md")); err == nil {
		for _, m := range ParseDependencyList(string(data), "mcps") {
			if !seen[m] {
				seen[m] = true
				result = append(result, m)
			}
		}
	}
	// Transitive MCPs from resolved skills
	for _, skill := range ResolveDependencies(swatRoot, squad) {
		skillMD := filepath.Join(swatRoot, "blueprints", "skills", skill, "SKILL.md")
		if data, err := os.ReadFile(skillMD); err == nil {
			for _, m := range ParseDependencyList(string(data), "mcps") {
				if !seen[m] {
					seen[m] = true
					result = append(result, m)
				}
			}
		}
	}
	return result
}
