package commander

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// Notify sends a notification to the user via OpenClaw Gateway /tools/invoke API.
// Falls back silently if environment variables are not configured.
func (c *Commander) Notify(message string) error {
	port := os.Getenv("OPENCLAW_GATEWAY_PORT")
	if port == "" {
		port = "18789"
	}
	token := os.Getenv("OPENCLAW_GATEWAY_TOKEN")
	if token == "" {
		return fmt.Errorf("OPENCLAW_GATEWAY_TOKEN not set")
	}
	target := os.Getenv("OPENCLAW_NOTIFY_TARGET")
	if target == "" {
		return fmt.Errorf("OPENCLAW_NOTIFY_TARGET not set")
	}
	channel := os.Getenv("OPENCLAW_NOTIFY_CHANNEL")
	if channel == "" {
		channel = "telegram"
	}

	payload := map[string]interface{}{
		"tool": "message",
		"args": map[string]interface{}{
			"action":  "send",
			"channel": channel,
			"target":  target,
			"message": message,
		},
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
