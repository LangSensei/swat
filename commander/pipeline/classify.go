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
		"You are a task classifier and enricher. "+
			"Read OPERATION.md in the current directory for the task. "+
			"Read all MANIFEST.md files under %s (skip _framework) to find available squads. "+
			"Read past operations under %s for historical context. "+
			"Your job: "+
			"After reading all context, update OPERATION.md in a SINGLE edit operation containing ALL of the following changes: "+
			"1. Set the 'squad' field in frontmatter to the best matching squad. "+
			"2. Set the 'references' field to relevant historical operations as [{type: \"operation\", value: \"path\"}]. "+
			"3. Replace the `[CLASSIFY: ...]` placeholder in ### Context with historical context, related operation findings, and key metrics. Keep the ## Assignment text intact. "+
			"CRITICAL: All three changes MUST be applied in one single file write. Do NOT split into multiple edit operations on the same file. "+
			"If no squad is a good fit for the task, leave the squad field empty. "+
			"Do NOT modify any other frontmatter fields besides 'squad' and 'references'.",
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
