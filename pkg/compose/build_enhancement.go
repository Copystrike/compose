/*
   Copyright 2020 Docker Compose CLI authors

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

package compose

import (
	"os"

	"github.com/compose-spec/compose-go/v2/types"
)

// EnhanceProjectWithBuildDependsOn reads the compose files and injects build.depends_on
// into the project's build configurations through extensions
func EnhanceProjectWithBuildDependsOn(project *types.Project) (*types.Project, error) {
	if len(project.ComposeFiles) == 0 {
		return project, nil
	}

	// Parse each compose file to extract build.depends_on
	buildDepsMap := make(map[string][]string)
	
	for _, filePath := range project.ComposeFiles {
		deps, err := extractBuildDependsOnFromFile(filePath)
		if err != nil {
			continue // Skip files that can't be parsed, don't fail the whole process
		}
		
		// Merge dependencies from this file
		for serviceName, serviceDeps := range deps {
			buildDepsMap[serviceName] = append(buildDepsMap[serviceName], serviceDeps...)
		}
	}

	// Inject dependencies into project services
	for serviceName, deps := range buildDepsMap {
		if service, exists := project.Services[serviceName]; exists && service.Build != nil {
			if service.Build.Extensions == nil {
				service.Build.Extensions = make(types.Extensions)
			}
			// Use "build-depends-on" to avoid conflicts with other extensions
			service.Build.Extensions["build-depends-on"] = deps
			project.Services[serviceName] = service
		}
	}

	return project, nil
}

// extractBuildDependsOnFromFile parses a compose file and extracts build.depends_on
func extractBuildDependsOnFromFile(filePath string) (map[string][]string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return parseBuildDependsOnFromYAML(content)
}

// getBuildDependsOnEnhanced extracts build.depends_on from enhanced project
func getBuildDependsOnEnhanced(service types.ServiceConfig) []string {
	if service.Build == nil {
		return nil
	}

	// Check for enhanced build-depends-on first
	if dependsOnRaw, ok := service.Build.Extensions["build-depends-on"]; ok {
		switch dependsOn := dependsOnRaw.(type) {
		case []interface{}:
			var deps []string
			for _, dep := range dependsOn {
				if depStr, ok := dep.(string); ok {
					deps = append(deps, depStr)
				}
			}
			return deps
		case []string:
			return dependsOn
		case string:
			return []string{dependsOn}
		}
	}

	// Fall back to original extension parsing
	return getBuildDependsOn(service)
}