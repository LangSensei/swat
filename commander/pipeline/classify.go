package pipeline

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/LangSensei/swat/commander/deps"
	"github.com/LangSensei/swat/commander/layout"
	"github.com/LangSensei/swat/commander/notify"
	"github.com/LangSensei/swat/commander/operation"
	"github.com/LangSensei/swat/commander/platform"
	"github.com/LangSensei/swat/commander/runtime"
)

// Classify runs the LLM-based classify+enrich step on an unclassified operation.
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
			"1. Choose the best squad and update the 'squad' field in OPERATION.md frontmatter. "+
			"2. If you find relevant historical operations, add them to the 'references' field as [{type: \"operation\", value: \"path\"}]. "+
			"3. Write enrichment to the `### Context` section (under ## Assignment). Keep the ## Assignment text intact. Write historical context, related operation findings, and key metrics into ### Context, replacing the `[CLASSIFY: ...]` placeholder. "+
			"If no squad is a good fit for the task, leave the squad field empty. "+
			"Do NOT modify any other frontmatter fields besides 'squad' and 'references'.",
		filepath.Join(layout.BlueprintDir(), "squads"),
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
		return nil, "", fmt.Errorf("start classify copilot: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		logFile.Close()
		log.Printf("[classify] %s: classify copilot exited with error: %v", op.OperationID, err)
	} else {
		logFile.Close()
		log.Printf("[classify] %s: classify copilot completed successfully", op.OperationID)
	}

	reloaded, err := operation.LoadUnclassified(op.OperationID)
	if err != nil {
		return nil, "", fmt.Errorf("reload after classify: %v", err)
	}
	log.Printf("[classify] %s: classify result — squad=%q", op.OperationID, reloaded.Squad)

	if reloaded.Squad == "" {
		squads := listSquadSummaries()
		if notifier != nil {
			notifier.Notify(fmt.Sprintf("Task could not be classified — no matching squad found.\n\nOperation: %s\nBrief: %s\n\nInstalled squads:\n%s", op.OperationID, op.Brief, squads))
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

func listSquadSummaries() string {
	entries, err := os.ReadDir(filepath.Join(layout.BlueprintDir(), "squads"))
	if err != nil {
		return "(none installed)"
	}
	var lines []string
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "_framework" {
			continue
		}
		name := entry.Name()
		desc := "(no description)"
		manifestPath := filepath.Join(layout.BlueprintSquadDir(name), "MANIFEST.md")
		if data, err := os.ReadFile(manifestPath); err == nil {
			if d := deps.ExtractFrontmatterField(string(data), "description"); d != "" {
				desc = d
			}
		}
		lines = append(lines, fmt.Sprintf("• %s — %s", name, desc))
	}
	if len(lines) == 0 {
		return "(none installed)"
	}
	return strings.Join(lines, "\n")
}
