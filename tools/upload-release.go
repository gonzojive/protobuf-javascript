package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var (
	tagFlag    = flag.String("tag", "", "The Git tag to create the release for")
	dryRunFlag = flag.Bool("dry-run", false, "Skip creating the GitHub release")
	draftFlag  = flag.Bool("draft", true, "Whether to create the release in a draft state.")
)

func runCommand(cmd *exec.Cmd) error {
	fmt.Printf("Running command: %s\n", strings.Join(cmd.Args, " "))
	cmd.Stdout = os.Stdout
	stdErr := bytes.Buffer{}
	cmd.Stderr = &stdErr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command failed: %w; command output: %s", err, string(stdErr.Bytes()))
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
	sourceZipPath := tempDir + "/protobuf-javascript-" + tag + ".zip"
	if err := runCommand(exec.Command("git", "archive", "--format", "zip",
		"--output", sourceZipPath,
		"--prefix", "protobuf-javascript-"+tag+"/",
		tag)); err != nil {
		return fmt.Errorf("Failed to archive code: %v", err)
	}

	createRelease := func(dryRun bool) error {
		releaseArgs := []string{
			"gh",
			"--repo", "gonzojive/protobuf-javascript",
			"release", "create",
			tag,
			sourceZipPath,
			"--verify-tag",
			"--title", tag,
			"--notes", "experimental version with bzlmod support.",
			"--prerelease",
		}
		if *draftFlag {
			releaseArgs = append(releaseArgs, "--draft")
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
	if err := createRelease(*dryRunFlag); err != nil {
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
