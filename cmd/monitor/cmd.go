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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"github.com/notaryproject/ratify-containerd/pkg/models"
	"github.com/notaryproject/ratify-containerd/pkg/shared"
)

const (
	// prefix of ConfigMap names to watch
	configMapPrefix = "scoped-config-"

	// namespace to watch ConfigMaps in， TODO: make this configurable
	namespace = "default"
)

func main() {
	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		logrus.Errorf("failed to build kubeconfig in %q: %v", kubeconfig, err)
		logrus.Info("Attempting to use in-cluster configuration...")
		config, err = clientcmd.BuildConfigFromFlags("", "")
		if err != nil {
			logrus.Errorf("failed to build in-cluster kubeconfig: %v", err)
			return
		}
	}
	logrus.Infof("Using kubeconfig: %s", kubeconfig)

	// create kubernetes clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logrus.Errorf("failed to create kubernetes clientset: %v", err)
		return
	}

	// watch ConfigMaps and write to shared volume
	for {
		err := processConfigMaps(clientset)
		if err != nil {
			// log the error and continue processing
			logrus.Errorf("Error processing ConfigMaps: %v", err)
		}
		time.Sleep(10 * time.Second) // wait before checking again
	}
}

// state tracking
var lastConfigMapNames = make(map[string]struct{})
var lastConfigMapVersions = make(map[string]string)

func processConfigMaps(clientset *kubernetes.Clientset) error {
	// list ConfigMaps in the provided namespace
	configMaps, err := clientset.CoreV1().ConfigMaps(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list ConfigMaps: %v", err)
	}

	currentNames := make(map[string]struct{})
	currentVersions := make(map[string]string)
	scopedConfigMaps := make([]*corev1.ConfigMap, 0, len(configMaps.Items))
	for _, cm := range configMaps.Items {
		if strings.HasPrefix(cm.Name, configMapPrefix) {
			currentNames[cm.Name] = struct{}{}
			currentVersions[cm.Name] = cm.ResourceVersion
			scopedConfigMaps = append(scopedConfigMaps, &cm)
		}
	}

	// detect added/removed ConfigMaps
	nameChanged := false
	if len(currentNames) != len(lastConfigMapNames) {
		logrus.Info("Quantity of ConfigMaps changed")
		nameChanged = true
	} else {
		for name := range currentNames {
			if _, ok := lastConfigMapNames[name]; !ok {
				logrus.Infof("ConfigMap %s added", name)
				nameChanged = true
				break
			}
		}
	}

	// detect content changes if names didn't change
	contentChanged := false
	if !nameChanged {
		for name, version := range currentVersions {
			if lastConfigMapVersions[name] != version {
				logrus.Infof("ConfigMap %s content changed (version %s -> %s)", name, lastConfigMapVersions[name], version)
				contentChanged = true
				break
			}
		}
	}

	if nameChanged || contentChanged {
		// on first run or if anything changed, process and write result
		logrus.Info("ConfigMap set or content changed, processing...")

		// process and write ConfigMap data to shared volume
		err := writeConfigMapsToSharedVolume(scopedConfigMaps)
		if err != nil {
			return fmt.Errorf("failed to write ConfigMaps to shared volume: %v", err)
		}

		logrus.Infof("Processing %d ConfigMaps", len(scopedConfigMaps))
		logrus.Infof("Wrote processed config map data to shared volume")
		// update state
		lastConfigMapNames = currentNames
		lastConfigMapVersions = currentVersions
	}
	// watch changes to ConfigMaps
	return nil
}

// writeFile writes the contents to the file at filePath
// it creates the directory if it does not exist
func writeFile(filePath string, contents string) error {
	// ensure the directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	err := os.WriteFile(filePath, []byte(contents), 0644)
	if err != nil {
		logrus.Errorf("failed writing to file %s: %v", filePath, err)
		return err
	}
	logrus.Infof("successfully wrote to file %s", filePath)
	return nil
}

// cleanupTempFiles removes all *.tmp files from the specified directory
func cleanupTempFiles(dirPath string) error {
	pattern := filepath.Join(dirPath, "*.tmp")
	tmpFiles, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to glob temp files: %v", err)
	}

	for _, tmpFile := range tmpFiles {
		if err := os.Remove(tmpFile); err != nil {
			logrus.Warnf("Failed to remove temp file %s: %v", tmpFile, err)
			// Continue removing other files even if one fails
		} else {
			logrus.Infof("Removed temp file: %s", tmpFile)
		}
	}

	return nil
}

// writeConfigToSharedVolume atomically writes configuration to shared volume
// This function ensures atomic writes by using temporary files
func writeConfigToSharedVolume(configJSON string) error {
	if err := cleanupTempFiles(shared.SharedVolumePath); err != nil {
		logrus.Warnf("Failed to cleanup temp files: %v", err)
		// Continue with write operation even if cleanup fails
	}

	tempFilePath := filepath.Join(shared.SharedVolumePath, shared.ScopedConfigFileName+".tmp")
	if err := writeFile(tempFilePath, configJSON); err != nil {
		return fmt.Errorf("failed to write to temp file: %v", err)
	}

	configMapPath := filepath.Join(shared.SharedVolumePath, shared.ScopedConfigFileName)
	if err := os.Rename(tempFilePath, configMapPath); err != nil {
		// Clean up temp file on failure
		os.Remove(tempFilePath)
		return fmt.Errorf("failed to rename temp file to final file: %v", err)
	}

	logrus.Infof("Successfully wrote configuration to %s", configMapPath)
	return nil
}

// writeConfigMapsToSharedVolume writes ConfigMap data to the shared volume
func writeConfigMapsToSharedVolume(configMaps []*corev1.ConfigMap) error {
	// collect all scopes from ConfigMaps
	allScopes := make([]string, 0)

	for _, cm := range configMaps {
		// process each ConfigMap's data
		for key, value := range cm.Data {
			logrus.Infof("Processing ConfigMap %s, key: %s", cm.Name, key)

			// try to parse JSON data as ScopedConfig
			var config models.ScopedConfig
			if err := json.Unmarshal([]byte(value), &config); err != nil {
				logrus.Warnf("Failed to parse JSON from ConfigMap %s, key %s: %v", cm.Name, key, err)
				continue
			}

			// collect scopes from this ConfigMap
			allScopes = append(allScopes, config.Scopes...)
		}
	}

	// convert to optimized format for efficient matching
	optimizedConfig := models.ToOptimizedFromScopes(allScopes)
	logrus.Infof("Collected %d scopes, deduplicated to %d unique scopes", len(allScopes), len(optimizedConfig.ScopeMap))

	// marshal to JSON
	configJSON, err := json.MarshalIndent(optimizedConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal combined config to JSON: %v", err)
	}

	// write to shared volume atomically
	if err := writeConfigToSharedVolume(string(configJSON)); err != nil {
		return fmt.Errorf("failed to write config to shared volume: %v", err)
	}

	logrus.Infof("Successfully wrote combined configuration to shared volume with %d unique scopes", len(optimizedConfig.ScopeMap))
	return nil
}
