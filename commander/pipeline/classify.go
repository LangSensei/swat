package pipeline

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/LangSensei/swat/commander/layout"
	"github.com/LangSensei/swat/commander/notify"
	"github.com/LangSensei/swat/commander/operation"
	"github.com/LangSensei/swat/commander/platform"
	"github.com/LangSensei/swat/commander/runtime"
	"github.com/LangSensei/swat/commander/squads"
)

// Classify runs the LLM-based operation classifier on an unclassified operation.
func Classify(rt runtime.RuntimeAdapter, op *operation.Operation, notifier notify.Notifier) (*operation.Operation, string, error) {
	unclassifiedDir := layout.UnclassifiedOperationDir(op.OperationID)

	if err := rt.PrepareWorkspace(unclassifiedDir, runtime.PhaseClassify); err != nil {
		return nil, "", fmt.Errorf("prepare workspace (classify): %v", err)
	}

	prompt := fmt.Sprintf(
		"You are an Operation Classifier. Your job is to route an operation to the right squad and enrich it with relevant context.\n\n"+
			"## Step 1: Read the operation\n"+
			"Read OPERATION.md for the brief (H1 title) and details (## Assignment section).\n\n"+
			"## Step 2: Read available squads\n"+
			"Read each MANIFEST.md under %s (skip _framework).\n"+
			"For each squad, note: name, domain, description, skills.\n\n"+
			"## Step 3: Match\n"+
			"Choose the squad whose domain and description best matches the operation.\n"+
			"Decision criteria (in priority order):\n"+
			"1. Domain match — operation subject falls within squad's stated domain\n"+
			"2. Skill match — operation requires skills the squad has\n"+
			"3. Specificity — prefer more specific squad over general one\n\n"+
			"If two squads tie, choose the one with more relevant historical operations.\n"+
			"If no squad fits, leave squad field empty.\n\n"+
			"## Step 4: Find references\n"+
			"Scan OPERATION.md files from the matched squad's operations/ directory (%s).\n"+
			"- First pass: read the frontmatter (status, summary) and the H1 title (brief) of all operations\n"+
			"- Second pass: for the 5-10 most relevant completed operations, read full content\n"+
			"- Add the most valuable ones as references (up to 10)\n\n"+
			"## Step 5: Enrich context\n"+
			"In the ### Context section, write:\n"+
			"- Why this squad was chosen (1 sentence)\n"+
			"- Key findings from referenced operations that are relevant to THIS operation\n"+
			"- Any data points or metrics the operator should know before starting\n\n"+
			"## Output\n"+
			"Update OPERATION.md frontmatter: squad and references fields only.\n"+
			"Replace the [CLASSIFY: ...] placeholder with your enrichment.\n"+
			"Do NOT modify any other frontmatter fields.",
		layout.BlueprintSquadsDir(),
		layout.SquadsDir(),
	)

	cmd := rt.BuildCommand(prompt, unclassifiedDir)

	logPath := filepath.Join(unclassifiedDir, "classify.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		return nil, "", fmt.Errorf("create classify log: %v", err)
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		logFile.Close()
		return nil, "", fmt.Errorf("start operation classifier: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		logFile.Close()
		log.Printf("[classify] %s: operation classifier exited with error: %v", op.OperationID, err)
	} else {
		logFile.Close()
		log.Printf("[classify] %s: operation classifier completed successfully", op.OperationID)
	}

	reloaded, err := operation.Load("_unclassified", op.OperationID)
	if err != nil {
		return nil, "", fmt.Errorf("reload after classify: %v", err)
	}
	log.Printf("[classify] %s: classify result — squad=%q", op.OperationID, reloaded.Squad)

	if reloaded.Squad == "" {
		summaries := squads.ListSummaries()
		if notifier != nil {
			notifier.Notify(fmt.Sprintf("Task could not be classified — no matching squad found.\n\nOperation: %s\nBrief: %s\n\nInstalled squads:\n%s", op.OperationID, op.Brief, summaries))
		}
		return nil, "", fmt.Errorf("classify failed: no squad assigned")
	}

	manifestPath := filepath.Join(layout.BlueprintSquadDir(reloaded.Squad), "MANIFEST.md")
	if !platform.FileExists(manifestPath) {
		if notifier != nil {
			notifier.Notify(fmt.Sprintf("Task classified to squad '%s' which is not installed.\n\nOperation: %s\nBrief: %s", reloaded.Squad, op.OperationID, op.Brief))
		}
		return nil, "", fmt.Errorf("classify assigned unknown squad: %s", reloaded.Squad)
	}

	destDir := layout.OperationDir(reloaded.Squad, op.OperationID)
	if err := os.MkdirAll(filepath.Dir(destDir), 0755); err != nil {
		return nil, "", fmt.Errorf("create squad dir: %v", err)
	}
	if err := os.Rename(unclassifiedDir, destDir); err != nil {
		return nil, "", fmt.Errorf("move to squad: %v", err)
	}

	return reloaded, destDir, nil
}
