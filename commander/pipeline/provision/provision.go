package provision

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/LangSensei/swat/commander/deps"
	"github.com/LangSensei/swat/commander/layout"
	"github.com/LangSensei/swat/commander/operation"
	"github.com/LangSensei/swat/commander/runtime"
)

// Run copies squad snapshot, skills, hooks, and protocol into the operation directory.
func Run(rt runtime.RuntimeAdapter, op *operation.Operation, opDir, swatRoot, runtimeName, notifyBackend string) error {
	squadBP := layout.SquadBlueprintDir(op.Squad)

	protocol, err := os.ReadFile(filepath.Join(layout.FrameworkDir(), "PROTOCOL.md"))
	if err != nil {
		return fmt.Errorf("read protocol: %w", err)
	}
	if err := rt.ComposeAgentFile(opDir, protocol); err != nil {
		return err
	}

	if err := rt.ComposeSquad(squadBP, opDir); err != nil {
		return err
	}

	resolvedMCPs := deps.ResolveMCPDependencies(swatRoot, op.Squad)
	servers := ComposeMCPConfig(swatRoot, runtimeName, notifyBackend, resolvedMCPs)
	if len(servers) > 0 {
		if err := rt.ComposeMCPConfig(opDir, servers); err != nil {
			return err
		}
	}

	resolvedSkills := deps.ResolveDependencies(swatRoot, op.Squad)
	if err := rt.ComposeSkills(layout.SkillsDir(), resolvedSkills, opDir); err != nil {
		return err
	}

	if err := rt.ComposeHooks(layout.SkillsDir(), resolvedSkills, opDir); err != nil {
		return err
	}

	if err := rt.PrepareWorkspace(opDir, runtime.PhaseOperate); err != nil {
		log.Printf("[provision] PrepareWorkspace (operate): %v", err)
	}

	return nil
}

// LaunchAgent starts a runtime agent process for the operation.
func LaunchAgent(rt runtime.RuntimeAdapter, op *operation.Operation, opDir string) error {
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

	if err := operation.Save(op); err != nil {
		cmd.Process.Kill()
		logFile.Close()
		return err
	}

	go func() {
		defer logFile.Close()
		cmd.Wait()
	}()

	return nil
}
