package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var (
	tagFlag = flag.String("tag", "", "The Git tag to create the release for")
)

func runCommand(cmd *exec.Cmd) error {
	fmt.Printf("Running command: %s\n", strings.Join(cmd.Args, " "))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command failed: %v", err)
	}
	return nil
}

func main() {
	if err := mainErr(); err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}
}

func mainErr() error {
	dryRunGithubPtr := flag.Bool("dry-run", false, "Skip creating the GitHub release")
	flag.Parse()

	var tag string
	if *tagFlag == "" {
		// Get the default tag from bazel module version
		depsNode, err := bazelModDeps("")
		if err != nil {
			return fmt.Errorf("Error getting default tag from bazel module: %v", err)
		}
		tag = depsNode.Version
	} else {
		tag = *tagFlag
	}

	// Create a temporary directory for the release artifacts
	tempDir, err := os.MkdirTemp("", "protobuf-javascript-release")
	if err != nil {
		return fmt.Errorf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create the Git tag
	if err := runCommand(exec.Command("git", "tag", tag, "--force")); err != nil {
		return fmt.Errorf("Failed to create Git tag: %v", err)
	}

	// Push the Git tag
	if err := runCommand(exec.Command("git", "push", "origin", "tag", tag, "--force")); err != nil {
		return fmt.Errorf("Failed to push Git tag: %v", err)
	}

	// Archive the code
	if err := runCommand(exec.Command("git", "archive", "--format", "zip", "--output", tempDir+"/protobuf-javascript-"+tag+".zip", "--prefix", "protobuf-javascript-"+tag+"/", tag)); err != nil {
		return fmt.Errorf("Failed to archive code: %v", err)
	}

	createRelease := func(tag string, tempDir string, dryRun bool) error {
		releaseArgs := []string{
			"gh", "release", "create",
			tag,
			tempDir + "/protobuf-javascript-" + tag + ".zip",
			"--verify-tag",
			"--title", tag,
			"--notes", "experimental version with bzlmod support.",
			"--draft", "--prerelease",
		}
		if !dryRun {
			return runCommand(exec.Command(releaseArgs[0], releaseArgs[1:]...))
		} else {
			fmt.Println("SKIPPING RELEASE -- GitHub release command would be:")
			fmt.Println("  " + strings.Join(releaseArgs, " "))
			return nil
		}
	}

	// Create the GitHub release
	if err := createRelease(tag, tempDir, *dryRunGithubPtr); err != nil {
		return fmt.Errorf("Failed to create GitHub release: %v", err)
	}

	return nil
}

type bazelDepsNode struct {
	Key                  string          `json:"key"`
	Name                 string          `json:"name"`
	Version              string          `json:"version"`
	Dependencies         []bazelDepsNode `json:"dependencies"`
	IndirectDependencies []bazelDepsNode `json:"indirectDependencies"`
	Cycles               []interface{}   `json:"cycles"`
	Root                 bool            `json:"root"`
}

func bazelModDeps(target string) (*bazelDepsNode, error) {
	cmd := exec.Command("bazel", "mod", "deps", "--output", "json")
	if target != "" {
		cmd.Args = append(cmd.Args, target)
	}
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("Error executing command: %w", err)
	}

	var depsNode bazelDepsNode
	err = json.Unmarshal(output, &depsNode)
	if err != nil {
		return nil, fmt.Errorf("Error parsing JSON: %w", err)
	}

	return &depsNode, nil
}
