package pipeline

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/LangSensei/swat/commander/layout"
	"github.com/LangSensei/swat/commander/operation"
	"github.com/LangSensei/swat/commander/platform"
	"github.com/LangSensei/swat/commander/runtime"
	"github.com/LangSensei/swat/commander/squads"
)

// SpawnClassify starts the LLM-based operation classifier as a non-blocking subprocess.
// It saves the classify PID and sets status to "classifying", then returns immediately.
func SpawnClassify(rt runtime.RuntimeAdapter, op *operation.Operation) error {
	unclassifiedDir := layout.UnclassifiedOperationDir(op.OperationID)

	if err := rt.PrepareWorkspace(unclassifiedDir, runtime.PhaseClassify); err != nil {
		log.Printf("[classify] %s: prepare workspace error: %v", op.OperationID, err)
		return fmt.Errorf("classify_spawn_failed")
	}

	prompt := buildClassifyPrompt()

	cmd := rt.BuildCommand(prompt, unclassifiedDir)

	logPath := filepath.Join(unclassifiedDir, "classify.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		log.Printf("[classify] %s: create log error: %v", op.OperationID, err)
		return fmt.Errorf("classify_spawn_failed")
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		logFile.Close()
		log.Printf("[classify] %s: spawn error: %v", op.OperationID, err)
		return fmt.Errorf("classify_spawn_failed")
	}

	now := time.Now().UTC()
	op.Status = "classifying"
	op.PID = cmd.Process.Pid
	op.DispatchedAt = &now

	if err := operation.Save(op); err != nil {
		cmd.Process.Kill()
		logFile.Close()
		log.Printf("[classify] %s: save state error: %v", op.OperationID, err)
		return fmt.Errorf("classify_spawn_failed")
	}

	go func() {
		defer logFile.Close()
		cmd.Wait()
	}()

	log.Printf("[classify] %s: classifier spawned (pid=%d)", op.OperationID, op.PID)
	return nil
}

// Advance completes classify and transitions an operation from classifying → active.
// Reloads OPERATION.md to get classifier output, validates squad, moves to squad dir,
// provisions, and launches the agent.
func Advance(rt runtime.RuntimeAdapter, op *operation.Operation, runtimeName, notifyName string) error {
	log.Printf("[classify] %s: classify result — squad=%q", op.OperationID, op.Squad)

	if op.Squad == "" {
		summaries := squads.ListSummaries()
		log.Printf("[classify] %s: no squad match. Installed squads:\n%s", op.OperationID, summaries)
		return fmt.Errorf("classify_no_squad")
	}

	manifestPath := filepath.Join(layout.BlueprintSquadDir(op.Squad), "MANIFEST.md")
	if !platform.FileExists(manifestPath) {
		log.Printf("[classify] %s: squad %q not installed (no MANIFEST.md)", op.OperationID, op.Squad)
		return fmt.Errorf("classify_squad_not_installed")
	}

	destDir := layout.OperationDir(op.Squad, op.OperationID)
	if err := os.MkdirAll(filepath.Dir(destDir), 0755); err != nil {
		log.Printf("[classify] %s: create squad dir error: %v", op.OperationID, err)
		return fmt.Errorf("classify_move_failed")
	}
	if err := os.Rename(layout.UnclassifiedOperationDir(op.OperationID), destDir); err != nil {
		log.Printf("[classify] %s: move to squad error: %v", op.OperationID, err)
		return fmt.Errorf("classify_move_failed")
	}

	if err := Provision(rt, op, destDir, runtimeName, notifyName); err != nil {
		log.Printf("[classify] %s: provision error: %v", op.OperationID, err)
		return fmt.Errorf("provision_failed")
	}

	if err := LaunchAgent(rt, op, destDir); err != nil {
		log.Printf("[classify] %s: launch error: %v", op.OperationID, err)
		return fmt.Errorf("launch_failed")
	}

	log.Printf("[scan] %s: launched successfully (squad=%s)", op.OperationID, op.Squad)
	return nil
}

// buildClassifyPrompt constructs the classifier system prompt.
func buildClassifyPrompt() string {
	return fmt.Sprintf(
		"You are an Operation Classifier. Your job is to route an operation to the right squad and enrich it with relevant context. "+
			"## Step 1: Read the operation "+
			"Read OPERATION.md for the brief (H1 title) and details (## Assignment section). "+
			"## Step 2: Read available squads "+
			"Read each MANIFEST.md under %s (skip _framework). "+
			"For each squad, note: name, domain, description, skills. "+
			"## Step 3: Match "+
			"Choose the squad whose domain and description best matches the operation. "+
			"Decision criteria (in priority order): "+
			"1. Domain match — operation subject falls within squad's stated domain "+
			"2. Skill match — operation requires skills the squad has "+
			"3. Specificity — prefer more specific squad over general one "+
			"If two squads tie, choose the one with more relevant historical operations. "+
			"If no squad fits, leave squad field empty. "+
			"## Step 4: Find references "+
			"Scan OPERATION.md files from the matched squad's operations/ directory (%s). "+
			"- First pass: read the frontmatter (status, summary) and the H1 title (brief) of all operations "+
			"- Second pass: for the 5-10 most relevant completed operations, read full content "+
			"- Add the most valuable ones as references (up to 10) "+
			"## Step 5: Enrich context "+
			"In the ### Context section, write: "+
			"- Why this squad was chosen (1 sentence) "+
			"- Key findings from referenced operations that are relevant to THIS operation "+
			"- Any data points or metrics the operator should know before starting "+
			"## Output "+
			"Update OPERATION.md in a SINGLE edit operation: set squad and references fields in frontmatter, and replace the [CLASSIFY: ...] placeholder with your enrichment. "+
			"CRITICAL: All changes to OPERATION.md MUST be applied in one single file write. Do NOT split into multiple edit operations on the same file. "+
			"Do NOT modify any other frontmatter fields.",
		layout.BlueprintSquadsDir(),
		layout.SquadsDir(),
	)
}
