package git

import (
	"fmt"
	"os/exec"
)

func GetGitDiff(repoPath string) string {
	cmd := exec.Command("git", "-C", repoPath, "diff")
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("Failed to get git diff: %v\n", err)
		return ""
	}
	return string(output)
}

func GetGitDiffBetweenBranches(repoPath, branch1, branch2 string) string {
	cmd := exec.Command("git", "-C", repoPath, "diff", branch1+".."+branch2)
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("Failed to get git diff between branches: %v\n", err)
		return ""
	}
	return string(output)
}

func GetGitLog(repoPath, branch1, branch2 string) string {
	cmd := exec.Command("git", "-C", repoPath, "log", "--oneline", branch1+".."+branch2)
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("Failed to get git log: %v\n", err)
		return ""
	}
	return string(output)
}

func BranchExists(repoPath, branchName string) bool {
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "--verify", branchName)
	err := cmd.Run()
	return err == nil
}
