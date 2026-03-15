package commander

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// Notify sends a notification directly to the user via OpenClaw Gateway message tool.
// Use for straightforward notifications (success, progress).
func (c *Commander) Notify(message string) error {
	return c.gatewayInvoke("message", map[string]interface{}{
		"action":  "send",
		"channel": getEnvOr("OPENCLAW_NOTIFY_CHANNEL", "telegram"),
		"target":  os.Getenv("OPENCLAW_NOTIFY_TARGET"),
		"message": message,
	})
}

// NotifySmart injects a message into the OpenClaw main session via cron wake,
// allowing the LLM to analyze and respond intelligently.
// Use for failures and situations that need LLM judgment (e.g., recommending actions).
func (c *Commander) NotifySmart(message string) error {
	return c.gatewayInvoke("cron", map[string]interface{}{
		"action": "wake",
		"text":   message,
		"mode":   "now",
	})
}

// gatewayInvoke calls OpenClaw Gateway /tools/invoke API
func (c *Commander) gatewayInvoke(tool string, args map[string]interface{}) error {
	port := getEnvOr("OPENCLAW_GATEWAY_PORT", "18789")
	token := os.Getenv("OPENCLAW_GATEWAY_TOKEN")
	if token == "" {
		return fmt.Errorf("OPENCLAW_GATEWAY_TOKEN not set")
	}

	if tool == "message" && os.Getenv("OPENCLAW_NOTIFY_TARGET") == "" {
		return fmt.Errorf("OPENCLAW_NOTIFY_TARGET not set")
	}

	payload := map[string]interface{}{
		"tool": tool,
		"args": args,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	url := fmt.Sprintf("http://127.0.0.1:%s/tools/invoke", port)
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("gateway returned %d", resp.StatusCode)
	}

	return nil
}

func getEnvOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
