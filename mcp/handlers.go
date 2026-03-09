package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
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
func Serve(cmdr *commander.Commander) error {
	srv := NewServer(cmdr)
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
					"version": "1.0.0",
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
	case "swat_list":
		return s.handleList(params.Arguments)
	case "swat_cancel":
		return s.handleCancel(params.Arguments)
	case "swat_squads":
		return s.handleSquads(params.Arguments)
	case "swat_schedule":
		return s.handleSchedule(params.Arguments)
	case "swat_install":
		return s.handleInstall(params.Arguments)
	case "swat_uninstall":
		return s.handleUninstall(params.Arguments)
	case "swat_browse":
		return s.handleBrowse(params.Arguments)
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
	squad, _ := args["squad"].(string)

	op, err := s.Commander.Dispatch(brief, details, squad)
	if err != nil {
		return toolResult{
			Content: []contentBlock{{Type: "text", Text: fmt.Sprintf("dispatch failed: %v", err)}},
			IsError: true,
		}
	}

	data, _ := json.MarshalIndent(op, "", "  ")
	return toolResult{
		Content: []contentBlock{{Type: "text", Text: string(data)}},
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

	result := map[string]interface{}{
		"counts":     counts,
		"operations": ops,
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return toolResult{
		Content: []contentBlock{{Type: "text", Text: string(data)}},
	}
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

func (s *Server) handleSchedule(args map[string]interface{}) toolResult {
	// TODO: create schedule entry
	return toolResult{
		Content: []contentBlock{{Type: "text", Text: "schedule not yet implemented"}},
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

	if err := s.Commander.Install(squad); err != nil {
		return toolResult{
			Content: []contentBlock{{Type: "text", Text: fmt.Sprintf("install failed: %v", err)}},
			IsError: true,
		}
	}

	return toolResult{
		Content: []contentBlock{{Type: "text", Text: fmt.Sprintf("squad %q installed successfully", squad)}},
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

func (s *Server) errorResponse(id interface{}, code int, msg string) *jsonrpcResponse {
	return &jsonrpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &rpcError{Code: code, Message: msg},
	}
}
