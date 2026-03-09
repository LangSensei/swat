package commander

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ReposDir returns the path to the persistent repos directory
func (c *Commander) ReposDir() string {
	return filepath.Join(c.SwatRoot, "repos")
}

// ensureRepo ensures a bare clone of the given repo exists at ~/.swat/repos/<name>/
// If it already exists, fetch latest. Returns the local repo path.
func (c *Commander) ensureRepo(repoURL, name string) (string, error) {
	repoDir := filepath.Join(c.ReposDir(), name)

	if dirExists(repoDir) {
		// Fetch latest
		cmd := exec.Command("git", "fetch", "--all", "--prune")
		cmd.Dir = repoDir
		if out, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("git fetch failed: %s: %w", string(out), err)
		}
		return repoDir, nil
	}

	// Clone as bare repo
	if err := os.MkdirAll(c.ReposDir(), 0755); err != nil {
		return "", err
	}
	cmd := exec.Command("git", "clone", "--bare", repoURL, repoDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git clone failed: %s: %w", string(out), err)
	}
	return repoDir, nil
}

// addWorktree creates a git worktree in the given directory on a new branch.
// The branch name is derived from the operation ID.
func (c *Commander) addWorktree(repoDir, worktreeDir, branchName string) error {
	// Create worktree with new branch based on origin/master
	cmd := exec.Command("git", "worktree", "add", "-b", branchName, worktreeDir, "origin/master")
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree add failed: %s: %w", string(out), err)
	}
	return nil
}

// removeWorktree removes a git worktree and prunes the branch
func (c *Commander) removeWorktree(repoDir, worktreeDir string) error {
	cmd := exec.Command("git", "worktree", "remove", "--force", worktreeDir)
	cmd.Dir = repoDir
	cmd.CombinedOutput() // best-effort
	return nil
}

// parseManifestRepo extracts the repo URL from a MANIFEST.md frontmatter field "repo:"
func parseManifestRepo(manifest string) string {
	for _, line := range strings.Split(manifest, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "repo:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "repo:"))
			val = strings.Trim(val, "\"'")
			return val
		}
	}
	return ""
}

// repoNameFromURL extracts repo name from a GitHub URL
func repoNameFromURL(url string) string {
	url = strings.TrimSuffix(url, ".git")
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "repo"
}
