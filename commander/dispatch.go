package commander

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Dispatch creates a new operation and launches it immediately
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

	// Launch immediately
	go func() {
		if err := c.launchOne(op); err != nil {
			reason := fmt.Sprintf("launch_failed: %v", err)
			c.Tracker.SetFailed(op.OperationID, reason)
		}
	}()

	return op, nil
}

// Cancel marks an operation as failed and kills the process if active
func (c *Commander) Cancel(opID string) error {
	// Kill process if tracked
	if tracked := c.Tracker.Get(opID); tracked != nil && tracked.PID > 0 {
		if p, err := os.FindProcess(tracked.PID); err == nil {
			p.Signal(os.Kill)
		}
	}
	return c.Tracker.SetFailed(opID, "cancelled_by_user")
}

// launchOne prepares the operation directory and starts a Copilot CLI process
func (c *Commander) launchOne(op *Operation) error {
	opDir := c.OperationDir(op.Squad, op.OperationID)

	if err := c.provision(op, opDir); err != nil {
		return fmt.Errorf("provision: %w", err)
	}

	prompt := "Begin operation. Read OPERATION.md for your task brief, then follow the protocol in AGENTS.md."

	cmd := exec.Command("copilot", "-p", prompt, "--yolo")
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

	now := time.Now().UTC()
	tracked := &TrackedOp{
		Squad:        op.Squad,
		PID:          cmd.Process.Pid,
		DispatchedAt: &now,
	}
	if err := c.Tracker.Track(op.OperationID, tracked); err != nil {
		cmd.Process.Kill()
		logFile.Close()
		return err
	}

	// Fire and forget — reap process to avoid zombie
	go func() {
		defer logFile.Close()
		cmd.Wait()
	}()

	return nil
}

// provision assembles AGENTS.md, copies skills and MCPs into the operation directory
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
