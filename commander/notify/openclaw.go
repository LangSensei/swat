package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// OpenClawNotifier sends notifications via the OpenClaw Gateway HTTP API.
type OpenClawNotifier struct {
	port    string
	token   string
	target  string
	channel string
}

// newOpenClawNotifier validates required env vars and returns an OpenClawNotifier.
func newOpenClawNotifier() (*OpenClawNotifier, error) {
	token := os.Getenv("OPENCLAW_GATEWAY_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("openclaw: OPENCLAW_GATEWAY_TOKEN is required")
	}
	target := os.Getenv("OPENCLAW_NOTIFY_TARGET")
	if target == "" {
		return nil, fmt.Errorf("openclaw: OPENCLAW_NOTIFY_TARGET is required")
	}

	port := os.Getenv("OPENCLAW_GATEWAY_PORT")
	if port == "" {
		port = "18789"
	}

	return &OpenClawNotifier{
		port:    port,
		token:   token,
		target:  target,
		channel: os.Getenv("OPENCLAW_NOTIFY_CHANNEL"),
	}, nil
}

// Notify sends a notification via the OpenClaw Gateway API.
// Retries up to 2 times with a 2-second delay between attempts.
func (o *OpenClawNotifier) Notify(message string) error {
	args := map[string]string{
		"action":  "send",
		"target":  o.target,
		"message": message,
	}
	if o.channel != "" {
		args["channel"] = o.channel
	}

	payload := map[string]interface{}{
		"tool": "message",
		"args": args,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("openclaw: marshal payload: %w", err)
	}

	url := fmt.Sprintf("http://127.0.0.1:%s/tools/invoke", o.port)

	const maxRetries = 2
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("openclaw: retry %d/%d...", attempt, maxRetries)
			time.Sleep(2 * time.Second)
		}

		lastErr = o.doPost(url, body)
		if lastErr == nil {
			return nil
		}
	}

	return fmt.Errorf("openclaw: failed after %d retries: %w", maxRetries, lastErr)
}

func (o *OpenClawNotifier) doPost(url string, body []byte) error {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+o.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("post: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	var result struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("parse response: %w (body: %s)", err, string(respBody))
	}
	if !result.OK {
		return fmt.Errorf("gateway returned ok:false (body: %s)", string(respBody))
	}

	return nil
}
