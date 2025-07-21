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

package models

import (
	"time"
)

// ScopedConfig represents the configuration structure for scoped repositories
type ScopedConfig struct {
	Version     string    `json:"version"`     // semver compliant schema version
	Scopes      []string  `json:"scopes"`      // repositories that need to be verified
	LastUpdated time.Time `json:"lastUpdated"` // timestamp of last update
}

// ScopedConfigOptimized represents an optimized structure for efficient scope matching
// Uses a map for O(1) scope lookups instead of O(n) array searches
type ScopedConfigOptimized struct {
	ScopeMap    map[string]bool `json:"scopeMap"`    // map for O(1) scope lookups
	LastUpdated time.Time       `json:"lastUpdated"` // timestamp of last update
}

// ToOptimized converts a ScopedConfig to ScopedConfigOptimized for efficient lookups
func (sc *ScopedConfig) ToOptimized() *ScopedConfigOptimized {
	scopeMap := make(map[string]bool, len(sc.Scopes))
	for _, scope := range sc.Scopes {
		scopeMap[scope] = true
	}

	return &ScopedConfigOptimized{
		ScopeMap:    scopeMap,
		LastUpdated: sc.LastUpdated,
	}
}

// ToOptimizedFromScopes creates a ScopedConfigOptimized from a slice of scopes.
func ToOptimizedFromScopes(scopes []string) *ScopedConfigOptimized {
	scopeMap := make(map[string]bool, len(scopes))
	for _, scope := range scopes {
		scopeMap[scope] = true
	}

	return &ScopedConfigOptimized{
		ScopeMap:    scopeMap,
		LastUpdated: time.Now().UTC(),
	}
}

// HasScope checks if a scope exists in the configuration (O(1) lookup)
func (sco *ScopedConfigOptimized) HasScope(scope string) bool {
	return sco.ScopeMap[scope]
}
