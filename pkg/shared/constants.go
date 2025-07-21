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

package shared

// TODO: make this configurable
const (
	// SharedVolumePath is the path to shared volume where configuration is stored
	SharedVolumePath = "/shared-data"

	// SharedVolumeMountPath is the mount path for the shared volume
	SharedVolumeMountPath = "/var"

	// ScopedConfigFileName is the name of the file to write the combined configuration to
	ScopedConfigFileName = "ratify-config.json"

	// RatifyConfigPath is the default path to ratify config file
	RatifyConfigPath = "/root/.ratify/bin/config.json"

	// RatifyBinPath is the default path to ratify binary
	RatifyBinPath = "/root/.ratify/bin/ratify"

	// DefaultHomeDir is the default home directory
	DefaultHomeDir = "/root"

	// VerifierLogFile is the path to the verifier log file
	VerifierLogFile = SharedVolumeMountPath + SharedVolumeMountPath + "ratify-verifier.log"
)
