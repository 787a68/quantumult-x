package io

import "os/exec"

func PushToReleases(branch string) error {
	cmds := [][]string{
		{"git", "config", "user.name", "github-actions[bot]"},
		{"git", "config", "user.email", "github-actions[bot]@users.noreply.github.com"},
		{"git", "checkout", "-B", branch},
		{"git", "add", "*.snippet"},
		{"git", "commit", "-m", "update snippets"},
		{"git", "push", "origin", branch, "--force"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		if out, err := cmd.CombinedOutput(); err != nil {
			return err
		} else {
			_ = out
		}
	}
	return nil
}