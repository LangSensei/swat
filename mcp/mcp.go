package mcp

import (
	"github.com/LangSensei/swat/commander"
	"github.com/LangSensei/swat/commander/notify"
)

// Server wraps the Commander as an MCP server
type Server struct {
	Commander *commander.Commander
	Notifier  notify.Notifier
	Version   string
}

// NewServer creates a new MCP server
func NewServer(cmdr *commander.Commander) *Server {
	return &Server{
		Commander: cmdr,
		Notifier:  cmdr.Notifier,
	}
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
			Description: "Dispatch a new task to a SWAT squad. Squad is auto-classified. Returns immediately; task runs in background.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"brief":   map[string]interface{}{"type": "string", "description": "Task description"},
					"details": map[string]interface{}{"type": "string", "description": "Additional details"},
				},
				"required": []string{"brief"},
			},
		},
		{
			Name:        "swat_ops",
			Description: "List SWAT operations with optional filters. Returns counts and matching operations.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"status": map[string]interface{}{"type": "string", "description": "Filter by status (queued/active/completed/failed)"},
					"since":  map[string]interface{}{"type": "string", "description": "Only return terminal ops after this RFC3339 timestamp"},
					"limit":  map[string]interface{}{"type": "integer", "description": "Max results to return (default 50)"},
					"offset": map[string]interface{}{"type": "integer", "description": "Skip first N results (default 0)"},
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
			Name:        "swat_schedule_create",
			Description: "Create a scheduled recurring task. Zero LLM cost.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"brief":     map[string]interface{}{"type": "string", "description": "Task description"},
					"details":   map[string]interface{}{"type": "string", "description": "Additional details"},
					"cron":      map[string]interface{}{"type": "string", "description": "Cron expression, 5-field: min hour dom month dow"},
					"timezone":  map[string]interface{}{"type": "string", "description": "IANA timezone, e.g. Asia/Shanghai (default: UTC)"},
					"immediate": map[string]interface{}{"type": "boolean", "description": "If true, trigger first run immediately (default: false)"},
				},
				"required": []string{"brief", "cron"},
			},
		},
		{
			Name:        "swat_schedules",
			Description: "List all scheduled tasks with next run times",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "swat_schedule_delete",
			Description: "Delete a scheduled task",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id": map[string]interface{}{"type": "string", "description": "Schedule ID"},
				},
				"required": []string{"id"},
			},
		},
		{
			Name:        "swat_squad_browse",
			Description: "List all squads available in the marketplace",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "swat_squad_install",
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
			Name:        "swat_squad_uninstall",
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
			Name:        "swat_squad_update",
			Description: "Update an installed squad to the latest marketplace version",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"squad": map[string]interface{}{"type": "string", "description": "Squad name to update"},
				},
				"required": []string{"squad"},
			},
		},
		{
			Name:        "swat_notify",
			Description: "Send a notification to the user.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"message": map[string]interface{}{"type": "string", "description": "Notification message to display"},
				},
				"required": []string{"message"},
			},
		},
	}
}
