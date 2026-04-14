package provision

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/LangSensei/swat/commander/layout"
)

// ComposeMCPConfig builds a map of MCP server entries from individual config files.
// runtimeName and notifyBackend are injected as --runtime and --notify flags
// into the "swat" server args, if present.
func ComposeMCPConfig(runtimeName, notifyBackend string, mcps []string) map[string]interface{} {
	mcpsDir := layout.BlueprintMCPsDir()
	servers := make(map[string]interface{})
	for _, name := range mcps {
		path := filepath.Join(mcpsDir, name+".json")
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var parsed interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			continue
		}

		// Inject --runtime and --notify into the swat server entry
		if name == "swat" {
			raw := strings.TrimSpace(string(data))
			raw = injectSwatArgs(raw, runtimeName, notifyBackend)
			if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
				continue
			}
		}
		servers[name] = parsed
	}
	return servers
}

// injectSwatArgs adds --runtime and --notify flags to a swat MCP server JSON config.
func injectSwatArgs(raw, runtimeName, notifyBackend string) string {
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &obj); err != nil {
		return raw
	}

	var existing []string
	if args, ok := obj["args"]; ok {
		if arr, ok := args.([]interface{}); ok {
			for _, a := range arr {
				if s, ok := a.(string); ok {
					existing = append(existing, s)
				}
			}
		}
	}

	if runtimeName != "" {
		existing = append(existing, "--runtime", runtimeName)
	}
	if notifyBackend != "" {
		existing = append(existing, "--notify", notifyBackend)
	}
	obj["args"] = existing

	out, err := json.Marshal(obj)
	if err != nil {
		return raw
	}
	return string(out)
}
