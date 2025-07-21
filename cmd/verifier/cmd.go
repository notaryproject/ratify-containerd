/*
Copyright The Ratify Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/notaryproject/ratify-containerd/pkg/models"
	"github.com/notaryproject/ratify-containerd/pkg/shared"
	"oras.land/oras-go/v2/registry"
)

// RatifyOutput represents the structure of ratify verify command output
type RatifyOutput struct {
	IsSuccess bool        `json:"isSuccess"`
	Result    interface{} `json:"result,omitempty"`
	Error     string      `json:"error,omitempty"`
}

var (
	name           string
	digest         string
	stdinMediaType string
)

func init() {
	// Define command line flags
	// Implemented after https://github.com/containerd/containerd/blob/061792f0ecf3684fb30a3a0eb006799b8c6638a7/pkg/imageverifier/bindir/bindir.go#L118-L122
	flag.StringVar(&name, "name", "", "Container image name (required)")
	flag.StringVar(&digest, "digest", "", "Container image digest (required)")
	flag.StringVar(&stdinMediaType, "stdin-media-type", "", "Stdin media type")
}

func main() {
	// Parse command line flags
	flag.Parse()

	// Validate required flags
	if name == "" {
		fmt.Printf("Error: -name flag is required\n")
		flag.Usage()
		os.Exit(1)
	}
	if digest == "" {
		fmt.Printf("Error: -digest flag is required\n")
		flag.Usage()
		os.Exit(1)
	}

	// Parse the registry reference from the name to extract repository
	ref, err := registry.ParseReference(name)
	if err != nil {
		fmt.Printf("Error: Invalid name format '%s': %v\n", name, err)
		os.Exit(1)
	}

	ref.Reference = ""

	if inScope, err := isRepositoryInScope(ref.String()); err != nil {
		fmt.Printf("Error checking repository scope: %v\n", err)
		os.Exit(1)
	} else if !inScope {
		fmt.Printf("Repository '%s' is not in scope. Skip checking.\n", ref.String())
		os.Exit(0)
	}

	// Set default paths
	configPath := shared.RatifyConfigPath

	// Check if ratify config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Println("ratify config file not found. failing open")
		os.Exit(0)
	}

	// Execute ratify verify command
	ratifyOutput, err := executeRatifyVerify(name, digest)
	if err != nil {
		fmt.Printf("Failed to execute ratify verify: %v\n", err)
		os.Exit(1)
	}

	// Parse and display the output
	var output RatifyOutput
	if err := json.Unmarshal([]byte(ratifyOutput), &output); err != nil {
		fmt.Printf("Failed to parse ratify output: %v\n", err)
		fmt.Println("Raw output:", ratifyOutput)
		os.Exit(1)
	}

	// Pretty print the JSON output
	prettyJSON, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Println("Raw output:", ratifyOutput)
	} else {
		fmt.Println(string(prettyJSON))
	}

	// Check if verification succeeded
	if output.IsSuccess {
		fmt.Println("ratify verification succeeded")
		os.Exit(0)
	} else {
		fmt.Println("ratify verification failed")
		os.Exit(1)
	}
}

func executeRatifyVerify(name string, digest string) (string, error) {
	// Set default paths
	configPath := shared.RatifyConfigPath
	ratifyBin := shared.RatifyBinPath

	// Construct the ratify verify command arguments
	args := []string{"verify", "-c", configPath, "-s", name, "--digest", digest}
	fmt.Printf("Executing command: %s %v\n", ratifyBin, args)
	// Create the command
	cmd := exec.Command(ratifyBin, args...)

	// Set environment variables
	cmd.Env = append(os.Environ(), fmt.Sprintf("HOME=%s", shared.DefaultHomeDir))

	// Execute the command and capture output
	output, err := cmd.Output()
	if err != nil {
		// ratify might exit with non-zero code even for valid verification failures
		// so we try to get the output from stderr as well
		if exitError, ok := err.(*exec.ExitError); ok {
			// Combine stdout and stderr
			combined := string(output) + string(exitError.Stderr)
			if len(combined) > 0 {
				fmt.Printf("Ratify command output (with stderr): %s\n", combined)
				return combined, nil
			}
		}
		return "", fmt.Errorf("ratify command failed: %w", err)
	}

	fmt.Printf("Ratify command output: %s\n", string(output))
	return string(output), nil
}

// isRepositoryInScope checks if a repository is in the scoped configuration file
// Returns true if the repository is found in the scope, false otherwise
func isRepositoryInScope(repository string) (bool, error) {
	// Construct the path to the scoped config file
	configFilePath := filepath.Join(shared.SharedVolumeMountPath, shared.SharedVolumePath, shared.ScopedConfigFileName)

	fmt.Printf("Checking repository scope for: %s\n", repository)
	fmt.Printf("Config file path: %s\n", configFilePath)

	// Check if the config file exists
	if _, err := os.Stat(configFilePath); err != nil {
		if os.IsNotExist(err) {
			// enforce verification if the config file does not exist
			return true, nil
		}
		return false, fmt.Errorf("failed to check scoped config file: %w", err)
	}

	// Read the config file
	data, err := os.ReadFile(configFilePath)
	if err != nil {
		return false, fmt.Errorf("failed to read scoped config file: %w", err)
	}

	// Parse the JSON using the existing model
	var config models.ScopedConfigOptimized
	if err := json.Unmarshal(data, &config); err != nil {
		return false, fmt.Errorf("failed to parse scoped config JSON: %w", err)
	}

	// Check if repository is in scope using the optimized HasScope method
	inScope := config.HasScope(repository)
	return inScope, nil
}
