package commander

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Dispatch creates a new operation
func (c *Commander) Dispatch(brief, details, squad string) (*Operation, error) {
	now := time.Now().UTC()
	op := &Operation{
		OperationID: GenerateOpID(),
		Brief:       brief,
		Details:     details,
		Squad:       squad,
		Status:      "queued",
		Source:      "user",
		CreatedAt:   now,
	}
	if err := c.SaveOperation(op); err != nil {
		return nil, err
	}

	// Launch immediately instead of waiting for BackgroundLoop
	go func() {
		if err := c.launchOne(op); err != nil {
			now := time.Now().UTC()
			reason := fmt.Sprintf("launch_failed: %v", err)
			op.Status = "failed"
			op.FailedAt = &now
			op.FailureReason = &reason
			c.SaveOperation(op)
		}
	}()

	return op, nil
}

// Cancel marks an operation as failed and kills the process if active
func (c *Commander) Cancel(opID string) error {
	op, err := c.findOperation(opID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	reason := "cancelled_by_user"

	if op.Status == "active" && op.PID > 0 {
		if p, err := os.FindProcess(op.PID); err == nil {
			p.Signal(os.Kill)
		}
	}

	op.Status = "failed"
	op.FailedAt = &now
	op.FailureReason = &reason
	return c.SaveOperation(op)
}

// Launch picks up queued operations and starts squad processes
func (c *Commander) Launch() {
	ops, err := c.ListOperations()
	if err != nil {
		return
	}

	activeCount := 0
	for _, op := range ops {
		if op.Status == "active" {
			activeCount++
		}
	}

	for _, op := range ops {
		if op.Status != "queued" {
			continue
		}
		if activeCount >= c.MaxConcurrent {
			break
		}
		if err := c.launchOne(op); err != nil {
			now := time.Now().UTC()
			reason := fmt.Sprintf("launch_failed: %v", err)
			op.Status = "failed"
			op.FailedAt = &now
			op.FailureReason = &reason
			c.SaveOperation(op)
			continue
		}
		activeCount++
	}
}

// launchOne prepares the operation directory and starts a Copilot CLI process
func (c *Commander) launchOne(op *Operation) error {
	opDir := c.OperationDir(op.Squad, op.OperationID)

	// Provision: assemble AGENTS.md, copy skills and INTEL
	if err := c.provision(op, opDir); err != nil {
		return fmt.Errorf("provision: %w", err)
	}

	// Build prompt
	prompt := "Read AGENTS.md first to understand who you are and how to operate. Then follow the Captain Protocol."

	// Start Copilot CLI
	cmd := exec.Command("copilot", "-p", prompt, "--allow-all-tools")
	cmd.Dir = opDir

	logPath := filepath.Join(opDir, "copilot.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		return fmt.Errorf("create log file: %w", err)
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		logFile.Close()
		return fmt.Errorf("start copilot: %w", err)
	}

	// Update operation status
	now := time.Now().UTC()
	op.Status = "active"
	op.PID = cmd.Process.Pid
	op.DispatchedAt = &now

	if err := c.SaveOperation(op); err != nil {
		cmd.Process.Kill()
		logFile.Close()
		return err
	}

	// Fire and forget — Scan() will check process status periodically
	go func() {
		defer logFile.Close()
		cmd.Wait()
	}()

	return nil
}

// provision assembles AGENTS.md, copies skills and INTEL into the operation directory
func (c *Commander) provision(op *Operation, opDir string) error {
	bpDir := filepath.Join(c.SwatRoot, "blueprints")
	squadBP := filepath.Join(bpDir, "squads", op.Squad)
	frameworkDir := filepath.Join(bpDir, "squads", "_framework")

	// Read squad manifest
	manifest, err := os.ReadFile(filepath.Join(squadBP, "MANIFEST.md"))
	if err != nil {
		return fmt.Errorf("read manifest for squad %q: %w", op.Squad, err)
	}

	// Read protocol template
	protocol, err := os.ReadFile(filepath.Join(frameworkDir, "PROTOCOL.md"))
	if err != nil {
		return fmt.Errorf("read protocol: %w", err)
	}

	// Assemble and write AGENTS.md
	agentsMD := assembleAgentsMD(string(manifest), string(protocol), op.Squad)
	if err := os.WriteFile(filepath.Join(opDir, "AGENTS.md"), []byte(agentsMD), 0644); err != nil {
		return fmt.Errorf("write AGENTS.md: %w", err)
	}

	// Ensure squad runtime dir and INTEL.md exist
	squadDir := c.SquadDir(op.Squad)
	squadIntel := filepath.Join(squadDir, "INTEL.md")
	if !fileExists(squadIntel) {
		templateIntel := filepath.Join(frameworkDir, "INTEL.md")
		if data, err := os.ReadFile(templateIntel); err == nil {
			initialized := strings.ReplaceAll(string(data), "{SQUAD_NAME}", op.Squad)
			os.MkdirAll(squadDir, 0755)
			os.WriteFile(squadIntel, []byte(initialized), 0644)
		}
	}

	// Compose .mcp.json from resolved MCP dependencies
	resolvedMCPs := c.resolveMCPDependencies(op.Squad)
	if len(resolvedMCPs) > 0 {
		mcpConfig := composeMCPConfig(c.SwatRoot, resolvedMCPs)
		if mcpConfig != "" {
			os.WriteFile(filepath.Join(opDir, ".mcp.json"), []byte(mcpConfig), 0644)
		}
	}

	// Copy skills (resolve dependencies recursively)
	skillsRoot := filepath.Join(c.SwatRoot, "blueprints", "skills")
	resolvedSkills := c.resolveDependencies(op.Squad)
	destSkillsDir := filepath.Join(opDir, ".github", "skills")
	for _, skill := range resolvedSkills {
		srcSkill := filepath.Join(skillsRoot, skill)
		if _, err := os.Stat(srcSkill); err == nil {
			copyDir(srcSkill, filepath.Join(destSkillsDir, skill))
		}
	}

	return nil
}

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

// resolveMCPDependencies collects all MCP names from protocol and manifest
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
	// Find "  skills: [...]" or "  mcps: [...]"
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

// findOperation locates an operation by ID across all squads
func (c *Commander) findOperation(opID string) (*Operation, error) {
	ops, err := c.ListOperations()
	if err != nil {
		return nil, err
	}
	for _, op := range ops {
		if op.OperationID == opID {
			return op, nil
		}
	}
	return nil, fmt.Errorf("operation %s not found", opID)
}

// ListSquads returns all installed squad blueprints
func (c *Commander) ListSquads() ([]map[string]string, error) {
	bpDir := filepath.Join(c.SwatRoot, "blueprints")
	entries, err := os.ReadDir(filepath.Join(bpDir, "squads"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var squads []map[string]string
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "_framework" {
			continue
		}
		info := map[string]string{"name": entry.Name()}
		manifestPath := filepath.Join(bpDir, "squads", entry.Name(), "MANIFEST.md")
		if data, err := os.ReadFile(manifestPath); err == nil {
			if desc := extractFrontmatterField(string(data), "description"); desc != "" {
				info["description"] = desc
			}
		}
		squads = append(squads, info)
	}
	return squads, nil
}

// --- helpers ---

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

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func fileContains(path, substr string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), substr)
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}
