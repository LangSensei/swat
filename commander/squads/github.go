package squads

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

const marketplaceAPI = "https://api.github.com/repos/LangSensei/swat-marketplace"

var (
	ghTokenOnce  sync.Once
	ghTokenValue string
)

func resolveGHToken() string {
	ghTokenOnce.Do(func() {
		if t := os.Getenv("GITHUB_TOKEN"); t != "" {
			ghTokenValue = t
			return
		}
		if out, err := exec.Command("gh", "auth", "token").Output(); err == nil {
			ghTokenValue = strings.TrimSpace(string(out))
		}
	})
	return ghTokenValue
}

func ghHTTPGet(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if token := resolveGHToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return http.DefaultClient.Do(req)
}

type ghEntry struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Path string `json:"path"`
}

func ghListDir(path string) ([]ghEntry, error) {
	url := fmt.Sprintf("%s/contents/%s", marketplaceAPI, path)
	resp, err := ghHTTPGet(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("path %q not found in marketplace", path)
	}
	if resp.StatusCode == 403 {
		return nil, fmt.Errorf("GitHub API rate limited (403) — install gh CLI or set GITHUB_TOKEN")
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var entries []ghEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func ghGetFile(path string) ([]byte, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/LangSensei/swat-marketplace/main/%s", path)
	resp, err := ghHTTPGet(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("file %q not found", path)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d fetching %s", resp.StatusCode, path)
	}

	return io.ReadAll(resp.Body)
}

func ghDownloadDir(remotePath, localDir string) error {
	entries, err := ghListDir(remotePath)
	if err != nil {
		return err
	}

	os.MkdirAll(localDir, 0755)

	for _, e := range entries {
		localPath := filepath.Join(localDir, e.Name)
		switch e.Type {
		case "file":
			data, err := ghGetFile(e.Path)
			if err != nil {
				return fmt.Errorf("download %s: %w", e.Path, err)
			}
			perm := os.FileMode(0644)
			if strings.HasSuffix(e.Name, ".sh") {
				perm = 0755
			}
			if err := os.WriteFile(localPath, data, perm); err != nil {
				return err
			}
		case "dir":
			if err := ghDownloadDir(e.Path, localPath); err != nil {
				return err
			}
		}
	}
	return nil
}
