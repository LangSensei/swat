package commander

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/LangSensei/swat/commander/runtime"
)

// Dispatch creates a new operation in _unclassified and starts async classify+enrich+launch
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
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[dispatch] PANIC in processOperation %s: %v", op.OperationID, r)
				reason := fmt.Sprintf("panic: %v", r)
				op.Status = "failed"
				now := time.Now().UTC()
				op.FailedAt = &now
				op.FailureReason = &reason
				c.SaveOperation(op)
			}
		}()
		c.processOperation(op)
	}()

	return op, nil
}

// processOperation runs classify+enrich via Copilot CLI, then provisions and launches
func (c *Commander) processOperation(op *Operation) {
	unclassifiedDir := c.UnclassifiedOperationDir(op.OperationID)
	log.Printf("[dispatch] processOperation started: %s", op.OperationID)

	// Validate squad exists before moving
	validateSquad := func(squad string) bool {
		manifestPath := filepath.Join(c.SwatRoot, "blueprints", "squads", squad, "MANIFEST.md")
		return fileExists(manifestPath)
	}

	// Run classify+enrich Copilot CLI
	prompt := fmt.Sprintf(
		"You are a task classifier and enricher. "+
			"Read OPERATION.md in the current directory for the task. "+
			"Read all MANIFEST.md files under %s (skip _framework) to find available squads. "+
			"Read past operations under %s for historical context. "+
			"Your job: "+
			"1. Choose the best squad and update the 'squad' field in OPERATION.md frontmatter. "+
			"2. If you find relevant historical operations, add them to the 'references' field as [{type: \"operation\", value: \"path\"}]. "+
			"3. Write enrichment to the `### Context` section (under ## Assignment). Keep the ## Assignment text intact. Write historical context, related operation findings, and key metrics into ### Context, replacing the `[CLASSIFY: ...]` placeholder. "+
			"If no squad is a good fit for the task, leave the squad field empty. "+
			"Do NOT modify any other frontmatter fields besides 'squad' and 'references'.",
		filepath.Join(c.SwatRoot, "blueprints", "squads"),
		filepath.Join(c.SwatRoot, "squads"),
	)

	// Create runtime adapter once for this operation
	rt, err := runtime.New()
	if err != nil {
		c.failOperation(op, fmt.Sprintf("init runtime: %v", err))
		return
	}

	// Prepare workspace for classify phase (lightweight, no git init)
	if err := rt.PrepareWorkspace(unclassifiedDir, runtime.PhaseClassify); err != nil {
		c.failOperation(op, fmt.Sprintf("prepare workspace (classify): %v", err))
		return
	}

	cmd := rt.BuildCommand(prompt, unclassifiedDir)

	logPath := filepath.Join(unclassifiedDir, "classify.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		c.failOperation(op, fmt.Sprintf("create classify log: %v", err))
		return
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		logFile.Close()
		log.Printf("[dispatch] %s: failed to start classify copilot: %v", op.OperationID, err)
		c.failOperation(op, fmt.Sprintf("start classify copilot: %v", err))
		return
	}
	if err := cmd.Wait(); err != nil {
		logFile.Close()
		log.Printf("[dispatch] %s: classify copilot exited with error: %v", op.OperationID, err)
	} else {
		logFile.Close()
		log.Printf("[dispatch] %s: classify copilot completed successfully", op.OperationID)
	}

	// Re-read OPERATION.md after classify
	reloaded, err := c.LoadUnclassifiedOperation(op.OperationID)
	if err != nil {
		log.Printf("[dispatch] %s: failed to reload after classify: %v", op.OperationID, err)
		c.failOperation(op, fmt.Sprintf("reload after classify: %v", err))
		return
	}
	log.Printf("[dispatch] %s: classify result — squad=%q", op.OperationID, reloaded.Squad)

	if reloaded.Squad == "" {
		log.Printf("[dispatch] %s: classify failed — no squad assigned", op.OperationID)
		c.failOperation(op, "classify failed: no squad assigned")
		squads := c.listSquadSummaries()
		c.Notify(fmt.Sprintf("⚠️ Task could not be classified — no matching squad found.\n\nOperation: %s\nBrief: %s\n\nInstalled squads:\n%s\n\nSuggestions: install a new squad from marketplace (`swat browse`) or rephrase the task.", op.OperationID, op.Brief, squads))
		return
	}

	// Validate squad exists in blueprints
	if !validateSquad(reloaded.Squad) {
		log.Printf("[dispatch] %s: unknown squad %q", op.OperationID, reloaded.Squad)
		c.failOperation(op, fmt.Sprintf("classify assigned unknown squad: %s", reloaded.Squad))
		c.Notify(fmt.Sprintf("⚠️ Task classified to squad '%s' which is not installed.\n\nOperation: %s\nBrief: %s\n\nTry `swat install %s` or `swat browse` to check availability.", reloaded.Squad, op.OperationID, op.Brief, reloaded.Squad))
		return
	}

	// Move from _unclassified to squad operations dir
	destDir := c.OperationDir(reloaded.Squad, op.OperationID)
	log.Printf("[dispatch] %s: moving to %s", op.OperationID, destDir)
	if err := os.MkdirAll(filepath.Dir(destDir), 0755); err != nil {
		log.Printf("[dispatch] %s: failed to create squad dir: %v", op.OperationID, err)
		c.failOperation(op, fmt.Sprintf("create squad dir: %v", err))
		return
	}
	if err := os.Rename(unclassifiedDir, destDir); err != nil {
		log.Printf("[dispatch] %s: failed to move: %v", op.OperationID, err)
		c.failOperation(op, fmt.Sprintf("move to squad: %v", err))
		return
	}

	// Provision and launch — reuse the same runtime adapter
	if err := c.provision(rt, reloaded, destDir); err != nil {
		log.Printf("[dispatch] %s: provision failed: %v", op.OperationID, err)
		c.failOperation(reloaded, fmt.Sprintf("provision: %v", err))
		return
	}

	if err := c.launchAgent(rt, reloaded, destDir); err != nil {
		log.Printf("[dispatch] %s: launch failed: %v", op.OperationID, err)
		c.failOperation(reloaded, fmt.Sprintf("launch: %v", err))
		return
	}
	log.Printf("[dispatch] %s: launched successfully (squad=%s)", op.OperationID, reloaded.Squad)
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

// launchAgent starts a runtime agent process for the operation
func (c *Commander) launchAgent(rt runtime.RuntimeAdapter, op *Operation, opDir string) error {

	prompt := "Begin operation. AGENTS.md contains your protocol. Read it first."
	cmd := rt.BuildCommand(prompt, opDir)

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

// provision copies squad snapshot, skills, hooks, and protocol into the operation directory
func (c *Commander) provision(rt runtime.RuntimeAdapter, op *Operation, opDir string) error {
	bpDir := filepath.Join(c.SwatRoot, "blueprints")
	squadBP := filepath.Join(bpDir, "squads", op.Squad)
	frameworkDir := filepath.Join(bpDir, "squads", "_framework")

	// Copy PROTOCOL.md → agent file (runtime-specific name, e.g. AGENTS.md for Copilot)
	protocol, err := os.ReadFile(filepath.Join(frameworkDir, "PROTOCOL.md"))
	if err != nil {
		return fmt.Errorf("read protocol: %w", err)
	}
	if err := rt.WriteAgentFile(opDir, protocol); err != nil {
		return err
	}

	// Copy squad blueprint snapshot to .squad/ (read-only reference)
	if err := rt.CopySquad(squadBP, opDir); err != nil {
		return err
	}

	// Compose MCP config from resolved MCP dependencies
	resolvedMCPs := c.resolveMCPDependencies(op.Squad)
	if len(resolvedMCPs) > 0 {
		mcpConfig := composeMCPConfig(c.SwatRoot, resolvedMCPs)
		if mcpConfig != "" {
			rt.WriteMCPConfig(opDir, mcpConfig)
		}
	}

	// Copy skills (resolve dependencies recursively)
	skillsRoot := filepath.Join(c.SwatRoot, "blueprints", "skills")
	resolvedSkills := c.resolveDependencies(op.Squad)
	if err := rt.CopySkills(skillsRoot, resolvedSkills, opDir); err != nil {
		return err
	}

	// Prepare workspace for operate phase (full setup, e.g. git init for hook discovery)
	rt.PrepareWorkspace(opDir, runtime.PhaseOperate)

	return nil
}
