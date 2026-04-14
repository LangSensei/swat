package provision

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/LangSensei/swat/commander/deps"
	"github.com/LangSensei/swat/commander/operation"
	"github.com/LangSensei/swat/commander/runtime"
)

// Run copies squad snapshot, skills, hooks, and protocol into the operation directory.
func Run(rt runtime.RuntimeAdapter, op *operation.Operation, opDir, swatRoot, runtimeName, notifyBackend string) error {
	bpDir := filepath.Join(swatRoot, "blueprints")
	squadBP := filepath.Join(bpDir, "squads", op.Squad)
	frameworkDir := filepath.Join(bpDir, "squads", "_framework")

	// Copy PROTOCOL.md → agent file (runtime-specific name, e.g. AGENTS.md for Copilot)
	protocol, err := os.ReadFile(filepath.Join(frameworkDir, "PROTOCOL.md"))
	if err != nil {
		return fmt.Errorf("read protocol: %w", err)
	}
	if err := rt.ComposeAgentFile(opDir, protocol); err != nil {
		return err
	}

	// Copy squad blueprint snapshot to .squad/ (read-only reference)
	if err := rt.ComposeSquad(squadBP, opDir); err != nil {
		return err
	}

	// Compose MCP config from resolved MCP dependencies
	resolvedMCPs := deps.ResolveMCPDependencies(swatRoot, op.Squad)
	servers := ComposeMCPConfig(swatRoot, runtimeName, notifyBackend, resolvedMCPs)
	if len(servers) > 0 {
		if err := rt.ComposeMCPConfig(opDir, servers); err != nil {
			return err
		}
	}

	// Copy skill content (resolve dependencies recursively)
	skillsRoot := filepath.Join(swatRoot, "blueprints", "skills")
	resolvedSkills := deps.ResolveDependencies(swatRoot, op.Squad)
	if err := rt.ComposeSkills(skillsRoot, resolvedSkills, opDir); err != nil {
		return err
	}

	// Copy runtime-specific hooks from resolved skills
	if err := rt.ComposeHooks(skillsRoot, resolvedSkills, opDir); err != nil {
		return err
	}

	// Prepare workspace for operate phase (full setup, e.g. git init for hook discovery)
	if err := rt.PrepareWorkspace(opDir, runtime.PhaseOperate); err != nil {
		log.Printf("[provision] PrepareWorkspace (operate): %v", err)
	}

	return nil
}

// LaunchAgent starts a runtime agent process for the operation.
func LaunchAgent(rt runtime.RuntimeAdapter, op *operation.Operation, opDir string, store *operation.Store) error {
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

	if err := store.Save(op); err != nil {
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
