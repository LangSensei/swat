package commander

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Dispatch creates a new operation in staging and starts async classify+enrich+launch
func (c *Commander) Dispatch(brief, details string) (*Operation, error) {
	now := time.Now().UTC()
	op := &Operation{
		OperationID: GenerateOpID(),
		Brief:       brief,
		Details:     details,
		Status:      "queued",
		Source:      "user",
		CreatedAt:   now,
	}
	if err := c.CreateOperation(op); err != nil {
		return nil, err
	}

	// Async: classify + enrich + provision + launch
	go c.processOperation(op)

	return op, nil
}

// processOperation runs classify+enrich via Copilot CLI, then provisions and launches
func (c *Commander) processOperation(op *Operation) {
	stagingDir := c.StagingOperationDir(op.OperationID)

	// Run classify+enrich Copilot CLI
	prompt := fmt.Sprintf(
		"You are a task classifier and enricher. "+
			"Read OPERATION.md in the current directory for the task. "+
			"Read all MANIFEST.md files under %s (skip _framework) to find available squads. "+
			"Read past operations under %s for historical context. "+
			"Your job:\n"+
			"1. Choose the best squad and update the 'squad' field in OPERATION.md frontmatter.\n"+
			"2. If you find relevant historical operations, add them to the 'references' field as [{type: \"operation\", value: \"path\"}].\n"+
			"3. Enrich the ## Assignment section with additional context from historical operations (past findings, key metrics, known issues). Keep the original task description intact, append enrichment below it.\n"+
			"Do NOT modify any other frontmatter fields besides 'squad' and 'references'.",
		filepath.Join(c.SwatRoot, "blueprints", "squads"),
		filepath.Join(c.SwatRoot, "squads"),
	)

	cmd := exec.Command("copilot", "-p", prompt, "--yolo")
	cmd.Dir = stagingDir

	logPath := filepath.Join(stagingDir, "classify.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		c.failOperation(op, fmt.Sprintf("create classify log: %v", err))
		return
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		logFile.Close()
		c.failOperation(op, fmt.Sprintf("start classify copilot: %v", err))
		return
	}
	cmd.Wait()
	logFile.Close()

	// Re-read OPERATION.md after classify
	reloaded, err := c.LoadStagingOperation(op.OperationID)
	if err != nil {
		c.failOperation(op, fmt.Sprintf("reload after classify: %v", err))
		return
	}

	if reloaded.Squad == "" {
		c.failOperation(op, "classify failed: no squad assigned")
		return
	}

	// Move from staging to squad operations dir
	destDir := c.OperationDir(reloaded.Squad, op.OperationID)
	if err := os.MkdirAll(filepath.Dir(destDir), 0755); err != nil {
		c.failOperation(op, fmt.Sprintf("create squad dir: %v", err))
		return
	}
	if err := os.Rename(stagingDir, destDir); err != nil {
		c.failOperation(op, fmt.Sprintf("move to squad: %v", err))
		return
	}

	// Update op with classified data (keep enriched brief from Copilot)
	reloaded.OperationID = op.OperationID

	// Provision and launch
	if err := c.provision(reloaded, destDir); err != nil {
		c.failOperation(reloaded, fmt.Sprintf("provision: %v", err))
		return
	}

	if err := c.launchCopilot(reloaded, destDir); err != nil {
		c.failOperation(reloaded, fmt.Sprintf("launch: %v", err))
		return
	}
}

// failOperation marks an operation as failed
func (c *Commander) failOperation(op *Operation, reason string) {
	now := time.Now().UTC()
	op.Status = "failed"
	op.FailedAt = &now
	op.FailureReason = &reason
	c.SaveOperation(op)
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

// launchCopilot starts a Copilot CLI process for the operation
func (c *Commander) launchCopilot(op *Operation, opDir string) error {
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
	op.Status = "active"
	op.PID = cmd.Process.Pid
	op.DispatchedAt = &now

	if err := c.SaveOperation(op); err != nil {
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
