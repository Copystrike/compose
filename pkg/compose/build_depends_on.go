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
	"context"
	"fmt"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/compose/v2/pkg/api"
)

// getBuildDependsOn extracts build.depends_on from a service's build configuration
func getBuildDependsOn(service types.ServiceConfig) []string {
	if service.Build == nil {
		return nil
	}

	// Try to get build.depends_on from extensions first (for explicit x-depends-on usage)
	if dependsOnRaw, ok := service.Build.Extensions["depends_on"]; ok {
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

	// Also try x-depends-on for explicit extension usage
	if dependsOnRaw, ok := service.Build.Extensions["x-depends-on"]; ok {
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
	
	return nil
}

// startBuildDependencies starts the services that a build depends on
func (s *composeService) startBuildDependencies(ctx context.Context, project *types.Project, serviceName string) ([]string, error) {
	service := project.Services[serviceName]
	buildDeps := getBuildDependsOnEnhanced(service)
	if len(buildDeps) == 0 {
		return nil, nil
	}

	// Validate that all dependencies exist
	for _, dep := range buildDeps {
		if _, exists := project.Services[dep]; !exists {
			return nil, fmt.Errorf("service %q depends on %q for build, but %q is not defined", serviceName, dep, dep)
		}
	}

	// Start dependency services
	upOptions := api.UpOptions{
		Create: api.CreateOptions{
			Services: buildDeps,
		},
		Start: api.StartOptions{
			Services: buildDeps,
		},
	}

	err := s.Up(ctx, project, upOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to start build dependencies for %q: %w", serviceName, err)
	}

	return buildDeps, nil
}

// stopBuildDependencies stops the services that were started for build dependencies
func (s *composeService) stopBuildDependencies(ctx context.Context, project *types.Project, servicesToStop []string) error {
	if len(servicesToStop) == 0 {
		return nil
	}

	stopOptions := api.StopOptions{
		Services: servicesToStop,
	}

	err := s.Stop(ctx, project.Name, stopOptions)
	if err != nil {
		return fmt.Errorf("failed to stop build dependencies: %w", err)
	}

	return nil
}