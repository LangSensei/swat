package mcp

import (
	"github.com/LangSensei/swat/commander"
)

// Server wraps the Commander as an MCP server
type Server struct {
	Commander *commander.Commander
}

// NewServer creates a new MCP server
func NewServer(cmdr *commander.Commander) *Server {
	return &Server{Commander: cmdr}
}

// ToolDef describes an MCP tool
type ToolDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// Tools returns the list of MCP tools this server provides
func (s *Server) Tools() []ToolDef {
	return []ToolDef{
		{
			Name:        "swat_dispatch",
			Description: "Dispatch a new task to a SWAT squad",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"brief":   map[string]interface{}{"type": "string", "description": "Task description"},
					"details": map[string]interface{}{"type": "string", "description": "Additional details"},
					"squad":   map[string]interface{}{"type": "string", "description": "Target squad (auto-classify if omitted)"},
				},
				"required": []string{"brief"},
			},
		},
		{
			Name:        "swat_list",
			Description: "List SWAT operations with optional filters. Returns counts and matching operations.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"status": map[string]interface{}{"type": "string", "description": "Filter by status (queued/active/completed/failed)"},
					"since":  map[string]interface{}{"type": "string", "description": "Only return terminal ops after this RFC3339 timestamp"},
				},
			},
		},
		{
			Name:        "swat_cancel",
			Description: "Cancel a SWAT operation",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"operation_id": map[string]interface{}{"type": "string", "description": "Operation ID to cancel"},
				},
				"required": []string{"operation_id"},
			},
		},
		{
			Name:        "swat_squads",
			Description: "List installed SWAT squads",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "swat_schedule",
			Description: "Create a scheduled task",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"brief":   map[string]interface{}{"type": "string", "description": "Task description"},
					"details": map[string]interface{}{"type": "string", "description": "Additional details"},
					"squad":   map[string]interface{}{"type": "string", "description": "Target squad"},
					"cron":    map[string]interface{}{"type": "string", "description": "Cron expression"},
				},
				"required": []string{"brief", "cron"},
			},
		},
		{
			Name:        "swat_install",
			Description: "Install a squad from the SWAT marketplace (auto-resolves skill and MCP dependencies)",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"squad": map[string]interface{}{"type": "string", "description": "Squad name to install"},
				},
				"required": []string{"squad"},
			},
		},
		{
			Name:        "swat_uninstall",
			Description: "Uninstall a squad and clean up orphaned dependencies",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"squad": map[string]interface{}{"type": "string", "description": "Squad name to uninstall"},
					"purge": map[string]interface{}{"type": "boolean", "description": "Also delete runtime data and operation history (default: false)"},
				},
				"required": []string{"squad"},
			},
		},
		{
			Name:        "swat_browse",
			Description: "List all squads available in the marketplace with install status",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	}
}
