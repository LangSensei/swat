package notify

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/LangSensei/swat/commander/layout"
)

// OpenClawNotifier sends notifications via the OpenClaw Gateway HTTP API.
type OpenClawNotifier struct {
	port    string
	token   string
	target  string
	channel string
}

// readOpenClawConfig reads ~/.openclaw/openclaw.json and extracts gateway
// port and auth token. Returns empty strings if the file is missing or invalid.
func readOpenClawConfig() (port, token string) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", ""
	}

	data, err := os.ReadFile(filepath.Join(home, ".openclaw", "openclaw.json"))
	if err != nil {
		return "", ""
	}

	var raw struct {
		Gateway struct {
			Port json.Number `json:"port"`
			Auth struct {
				Token string `json:"token"`
			} `json:"auth"`
		} `json:"gateway"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return "", ""
	}

	return raw.Gateway.Port.String(), raw.Gateway.Auth.Token
}

// readDotEnv reads ~/.swat/.env and returns the values of the requested keys.
// The file format is KEY=VALUE per line; blank lines and # comments are skipped.
func readDotEnv(keys ...string) map[string]string {
	result := make(map[string]string, len(keys))

	f, err := os.Open(filepath.Join(layout.Root(), ".env"))
	if err != nil {
		return result
	}
	defer f.Close()

	want := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		want[k] = struct{}{}
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if _, ok := want[key]; ok {
			result[key] = value
		}
	}

	return result
}

// newOpenClawNotifier builds an OpenClawNotifier from two sources:
//   - port + token from ~/.openclaw/openclaw.json
//   - target + channel from ~/.swat/.env (OPENCLAW_NOTIFY_TARGET, OPENCLAW_NOTIFY_CHANNEL)
//
// Each value has exactly one source. Returns an error if any is missing.
func newOpenClawNotifier() (*OpenClawNotifier, error) {
	port, token := readOpenClawConfig()
	env := readDotEnv("OPENCLAW_NOTIFY_TARGET", "OPENCLAW_NOTIFY_CHANNEL")
	target := env["OPENCLAW_NOTIFY_TARGET"]
	channel := env["OPENCLAW_NOTIFY_CHANNEL"]

	if port == "" {
		return nil, fmt.Errorf("openclaw: gateway port is required (configure gateway.port in ~/.openclaw/openclaw.json)")
	}
	if token == "" {
		return nil, fmt.Errorf("openclaw: gateway token is required (configure gateway.auth.token in ~/.openclaw/openclaw.json)")
	}
	if target == "" {
		return nil, fmt.Errorf("openclaw: notify target is required (set OPENCLAW_NOTIFY_TARGET in ~/.swat/.env)")
	}
	if channel == "" {
		return nil, fmt.Errorf("openclaw: notify channel is required (set OPENCLAW_NOTIFY_CHANNEL in ~/.swat/.env)")
	}

	return &OpenClawNotifier{
		port:    port,
		token:   token,
		target:  target,
		channel: channel,
	}, nil
}

// Notify sends a notification via the OpenClaw Gateway API.
// Retries up to 2 times with a 2-second delay between attempts.
func (o *OpenClawNotifier) Notify(opID string, message string) error {
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
