package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"time"

	"github.com/LangSensei/swat/commander"
)

// JSON-RPC types
type jsonrpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonrpcResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *rpcError   `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type callToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type toolResult struct {
	Content []contentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Serve starts the MCP server on stdin/stdout
func Serve(cmdr *commander.Commander, version string) error {
	srv := NewServer(cmdr)
	srv.Version = version
	reader := bufio.NewReader(os.Stdin)
	writer := os.Stdout

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("read error: %w", err)
		}

		var req jsonrpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			log.Printf("[mcp] invalid JSON: %v", err)
			continue
		}

		resp := srv.handleRequest(req)
		if resp != nil {
			data, _ := json.Marshal(resp)
			data = append(data, '\n')
			writer.Write(data)
		}
	}
}

func (s *Server) handleRequest(req jsonrpcRequest) *jsonrpcResponse {
	switch req.Method {
	case "initialize":
		return &jsonrpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]interface{}{
					"tools": map[string]interface{}{},
				},
				"serverInfo": map[string]interface{}{
					"name":    "swat",
					"version": s.Version,
				},
			},
		}

	case "notifications/initialized":
		return nil // no response for notifications

	case "tools/list":
		return &jsonrpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"tools": s.Tools(),
			},
		}

	case "tools/call":
		var params callToolParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return s.errorResponse(req.ID, -32602, "invalid params")
		}
		result := s.handleToolCall(params)
		return &jsonrpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  result,
		}

	default:
		return s.errorResponse(req.ID, -32601, fmt.Sprintf("method not found: %s", req.Method))
	}
}

func (s *Server) handleToolCall(params callToolParams) toolResult {
	switch params.Name {
	case "swat_dispatch":
		return s.handleDispatch(params.Arguments)
	case "swat_ops":
		return s.handleList(params.Arguments)
	case "swat_cancel":
		return s.handleCancel(params.Arguments)
	case "swat_squads":
		return s.handleSquads(params.Arguments)
	case "swat_schedule_create":
		return s.handleScheduleCreate(params.Arguments)
	case "swat_schedules":
		return s.handleScheduleList(params.Arguments)
	case "swat_schedule_delete":
		return s.handleScheduleDelete(params.Arguments)
	case "swat_squad_install":
		return s.handleInstall(params.Arguments)
	case "swat_squad_uninstall":
		return s.handleUninstall(params.Arguments)
	case "swat_squad_browse":
		return s.handleBrowse(params.Arguments)
	case "swat_squad_update":
		return s.handleUpdate(params.Arguments)
	case "swat_notify":
		return s.handleNotify(params.Arguments)
	default:
		return toolResult{
			Content: []contentBlock{{Type: "text", Text: fmt.Sprintf("unknown tool: %s", params.Name)}},
			IsError: true,
		}
	}
}

func (s *Server) handleDispatch(args map[string]interface{}) toolResult {
	brief, _ := args["brief"].(string)
	details, _ := args["details"].(string)

	op, err := s.Commander.Dispatch(brief, details)
	if err != nil {
		return toolResult{
			Content: []contentBlock{{Type: "text", Text: fmt.Sprintf("dispatch failed: %v", err)}},
			IsError: true,
		}
	}

	data, _ := json.MarshalIndent(op, "", "  ")
	return toolResult{
		Content: []contentBlock{{Type: "text", Text: fmt.Sprintf("Operation queued. Classify + enrich + launch running async.\n%s", string(data))}},
	}
}

func (s *Server) handleList(args map[string]interface{}) toolResult {
	ops, err := s.Commander.ListOperations()
	if err != nil {
		return toolResult{
			Content: []contentBlock{{Type: "text", Text: fmt.Sprintf("list failed: %v", err)}},
			IsError: true,
		}
	}

	// Compute counts before filtering
	counts := map[string]int{"queued": 0, "active": 0, "completed": 0, "failed": 0}
	for _, op := range ops {
		counts[op.Status]++
	}

	statusFilter, _ := args["status"].(string)
	if statusFilter != "" {
		var filtered []*commander.Operation
		for _, op := range ops {
			if op.Status == statusFilter {
				filtered = append(filtered, op)
			}
		}
		ops = filtered
	}

	// Filter by time: only return ops with completed_at or failed_at after "since"
	sinceStr, _ := args["since"].(string)
	if sinceStr != "" {
		if sinceTime, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			var filtered []*commander.Operation
			for _, op := range ops {
				if op.CompletedAt != nil && op.CompletedAt.After(sinceTime) {
					filtered = append(filtered, op)
				} else if op.FailedAt != nil && op.FailedAt.After(sinceTime) {
					filtered = append(filtered, op)
				} else if op.Status == "active" || op.Status == "queued" {
					filtered = append(filtered, op) // always include in-flight
				}
			}
			ops = filtered
		}
	}

	// Sort by time descending (most recent first)
	sort.Slice(ops, func(i, j int) bool {
		return opSortTime(ops[i]).After(opSortTime(ops[j]))
	})

	// Pagination: offset then limit (default limit=50)
	offset := 0
	if v, ok := args["offset"].(float64); ok && int(v) > 0 {
		offset = int(v)
	}
	limit := 50
	if v, ok := args["limit"].(float64); ok && int(v) > 0 {
		limit = int(v)
	}
	if offset > len(ops) {
		ops = nil
	} else {
		ops = ops[offset:]
		if limit < len(ops) {
			ops = ops[:limit]
		}
	}

	result := map[string]interface{}{
		"counts":     counts,
		"operations": ops,
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return toolResult{
		Content: []contentBlock{{Type: "text", Text: string(data)}},
	}
}

// opSortTime returns the most relevant timestamp for sorting (newest event first)
func opSortTime(op *commander.Operation) time.Time {
	if op.CompletedAt != nil {
		return *op.CompletedAt
	}
	if op.FailedAt != nil {
		return *op.FailedAt
	}
	if op.DispatchedAt != nil {
		return *op.DispatchedAt
	}
	return op.CreatedAt
}

func (s *Server) handleCancel(args map[string]interface{}) toolResult {
	opID, _ := args["operation_id"].(string)
	if opID == "" {
		return toolResult{
			Content: []contentBlock{{Type: "text", Text: "operation_id is required"}},
			IsError: true,
		}
	}

	if err := s.Commander.Cancel(opID); err != nil {
		return toolResult{
			Content: []contentBlock{{Type: "text", Text: fmt.Sprintf("cancel failed: %v", err)}},
			IsError: true,
		}
	}

	return toolResult{
		Content: []contentBlock{{Type: "text", Text: fmt.Sprintf("operation %s cancelled", opID)}},
	}
}

func (s *Server) handleSquads(args map[string]interface{}) toolResult {
	squads, err := s.Commander.ListSquads()
	if err != nil {
		return toolResult{
			Content: []contentBlock{{Type: "text", Text: fmt.Sprintf("list squads failed: %v", err)}},
			IsError: true,
		}
	}
	if len(squads) == 0 {
		return toolResult{
			Content: []contentBlock{{Type: "text", Text: "no squads installed"}},
		}
	}
	data, _ := json.MarshalIndent(squads, "", "  ")
	return toolResult{
		Content: []contentBlock{{Type: "text", Text: string(data)}},
	}
}

func (s *Server) handleScheduleCreate(args map[string]interface{}) toolResult {
	brief, _ := args["brief"].(string)
	details, _ := args["details"].(string)
	cronExpr, _ := args["cron"].(string)
	tz, _ := args["timezone"].(string)
	immediate, _ := args["immediate"].(bool)

	sched, err := s.Commander.CreateSchedule(brief, details, cronExpr, tz, immediate)
	if err != nil {
		return toolResult{
			Content: []contentBlock{{Type: "text", Text: fmt.Sprintf("schedule failed: %v", err)}},
			IsError: true,
		}
	}
	data, _ := json.MarshalIndent(sched, "", "  ")
	return toolResult{
		Content: []contentBlock{{Type: "text", Text: string(data)}},
	}
}

func (s *Server) handleScheduleList(args map[string]interface{}) toolResult {
	schedules, err := s.Commander.ListSchedules()
	if err != nil {
		return toolResult{
			Content: []contentBlock{{Type: "text", Text: fmt.Sprintf("list failed: %v", err)}},
			IsError: true,
		}
	}
	data, _ := json.MarshalIndent(schedules, "", "  ")
	return toolResult{
		Content: []contentBlock{{Type: "text", Text: string(data)}},
	}
}

func (s *Server) handleScheduleDelete(args map[string]interface{}) toolResult {
	id, _ := args["id"].(string)
	if id == "" {
		return toolResult{
			Content: []contentBlock{{Type: "text", Text: "id is required"}},
			IsError: true,
		}
	}
	if err := s.Commander.DeleteSchedule(id); err != nil {
		return toolResult{
			Content: []contentBlock{{Type: "text", Text: fmt.Sprintf("delete failed: %v", err)}},
			IsError: true,
		}
	}
	return toolResult{
		Content: []contentBlock{{Type: "text", Text: fmt.Sprintf("schedule %s deleted", id)}},
	}
}

func (s *Server) handleInstall(args map[string]interface{}) toolResult {
	squad, _ := args["squad"].(string)
	if squad == "" {
		return toolResult{
			Content: []contentBlock{{Type: "text", Text: "squad name is required"}},
			IsError: true,
		}
	}

	prereqs, err := s.Commander.Install(squad)
	if err != nil {
		return toolResult{
			Content: []contentBlock{{Type: "text", Text: fmt.Sprintf("install failed: %v", err)}},
			IsError: true,
		}
	}

	msg := fmt.Sprintf("squad %q installed successfully", squad)
	if len(prereqs) > 0 {
		msg += "\n\n⚠️ Prerequisites needed:"
		for _, p := range prereqs {
			msg += fmt.Sprintf("\n- skill %q requires setup: %s", p.Skill, p.Path)
		}
		msg += "\n\nPlease complete these prerequisites before dispatching tasks."
	}

	return toolResult{
		Content: []contentBlock{{Type: "text", Text: msg}},
	}
}

func (s *Server) handleUninstall(args map[string]interface{}) toolResult {
	squad, _ := args["squad"].(string)
	if squad == "" {
		return toolResult{
			Content: []contentBlock{{Type: "text", Text: "squad name is required"}},
			IsError: true,
		}
	}

	purge, _ := args["purge"].(bool)

	if err := s.Commander.Uninstall(squad, purge); err != nil {
		return toolResult{
			Content: []contentBlock{{Type: "text", Text: fmt.Sprintf("uninstall failed: %v", err)}},
			IsError: true,
		}
	}

	msg := fmt.Sprintf("squad %q uninstalled", squad)
	if purge {
		msg += " (runtime data purged)"
	}
	return toolResult{
		Content: []contentBlock{{Type: "text", Text: msg}},
	}
}

func (s *Server) handleBrowse(args map[string]interface{}) toolResult {
	results, err := s.Commander.Browse()
	if err != nil {
		return toolResult{
			Content: []contentBlock{{Type: "text", Text: fmt.Sprintf("browse failed: %v", err)}},
			IsError: true,
		}
	}
	if len(results) == 0 {
		return toolResult{
			Content: []contentBlock{{Type: "text", Text: "no squads available in marketplace"}},
		}
	}
	data, _ := json.MarshalIndent(results, "", "  ")
	return toolResult{
		Content: []contentBlock{{Type: "text", Text: string(data)}},
	}
}

func (s *Server) handleUpdate(args map[string]interface{}) toolResult {
	squad, _ := args["squad"].(string)
	if squad == "" {
		return toolResult{
			Content: []contentBlock{{Type: "text", Text: "squad name is required"}},
			IsError: true,
		}
	}

	if err := s.Commander.Update(squad); err != nil {
		return toolResult{
			Content: []contentBlock{{Type: "text", Text: fmt.Sprintf("update failed: %v", err)}},
			IsError: true,
		}
	}

	return toolResult{
		Content: []contentBlock{{Type: "text", Text: fmt.Sprintf("squad %q updated to latest version", squad)}},
	}
}

func (s *Server) errorResponse(id interface{}, code int, msg string) *jsonrpcResponse {
	return &jsonrpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &rpcError{Code: code, Message: msg},
	}
}

func (s *Server) handleNotify(args map[string]interface{}) toolResult {
	message, _ := args["message"].(string)
	if message == "" {
		return toolResult{
			Content: []contentBlock{{Type: "text", Text: "message is required"}},
			IsError: true,
		}
	}

	if s.Notifier == nil {
		return toolResult{
			Content: []contentBlock{{Type: "text", Text: "notification backend not configured"}},
			IsError: true,
		}
	}

	if err := s.Notifier.Notify(message); err != nil {
		return toolResult{
			Content: []contentBlock{{Type: "text", Text: fmt.Sprintf("notify failed: %v", err)}},
			IsError: true,
		}
	}

	return toolResult{
		Content: []contentBlock{{Type: "text", Text: "notification sent"}},
	}
}
