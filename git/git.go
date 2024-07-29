package git

import (
	"fmt"
	"os/exec"
)

func GetGitDiff(repoPath string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "diff")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git diff: %w", err)
	}
	return string(output), nil
}

func GetGitDiffBetweenBranches(repoPath, branch1, branch2 string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "diff", branch1+".."+branch2)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git diff between branches: %w", err)
	}
	return string(output), nil
}

func GetGitLog(repoPath, branch1, branch2 string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "log", "--oneline", branch1+".."+branch2)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git log: %w", err)
	}
	return string(output), nil
}

func BranchExists(repoPath, branchName string) (bool, error) {
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "--verify", branchName)
	err := cmd.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			// Branch doesn't exist
			if exitError.ExitCode() == 1 {
				return false, nil
			}
		}
		return false, fmt.Errorf("failed to check if branch exists: %w", err)
	}
	return true, nil
}
