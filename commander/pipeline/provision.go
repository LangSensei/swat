package pipeline

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
func Run(rt runtime.RuntimeAdapter, op *operation.Operation, opDir, runtimeName, notify string) error {
	squadBP := layout.BlueprintSquadDir(op.Squad)

	protocol, err := os.ReadFile(filepath.Join(layout.BlueprintFrameworkDir(), "PROTOCOL.md"))
	if err != nil {
		return fmt.Errorf("read protocol: %w", err)
	}
	if err := rt.ComposeAgentFile(opDir, protocol); err != nil {
		return err
	}

	if err := rt.ComposeSquad(squadBP, opDir); err != nil {
		return err
	}

	resolvedMCPs := deps.ResolveMCPDependencies(op.Squad)
	if err := rt.ComposeMCPConfig(opDir, runtimeName, notify, resolvedMCPs); err != nil {
		return err
	}

	resolvedSkills := deps.ResolveDependencies(op.Squad)
	if err := rt.ComposeSkills(layout.BlueprintSkillsDir(), resolvedSkills, opDir); err != nil {
		return err
	}

	if err := rt.ComposeHooks(layout.BlueprintSkillsDir(), resolvedSkills, opDir); err != nil {
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
