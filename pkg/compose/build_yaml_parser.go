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
	"fmt"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v3"
)

// ProjectWithBuildDependsOn parses build.depends_on from YAML and injects it into BuildConfig extensions
func ProjectWithBuildDependsOn(project *types.Project, configs []types.ConfigFile) (*types.Project, error) {
	if len(configs) == 0 {
		return project, nil
	}

	// Parse the raw config to extract build.depends_on
	for _, config := range configs {
		if config.Config == nil {
			continue
		}

		services, ok := config.Config["services"].(map[string]interface{})
		if !ok {
			continue
		}

		for serviceName, serviceConfig := range services {
			serviceMap, ok := serviceConfig.(map[string]interface{})
			if !ok {
				continue
			}

			buildConfig, ok := serviceMap["build"].(map[string]interface{})
			if !ok {
				continue
			}

			dependsOn, ok := buildConfig["depends_on"]
			if !ok {
				continue
			}

			// Inject the depends_on into the service's build extensions
			if service, exists := project.Services[serviceName]; exists && service.Build != nil {
				if service.Build.Extensions == nil {
					service.Build.Extensions = make(types.Extensions)
				}
				service.Build.Extensions["depends_on"] = dependsOn
				project.Services[serviceName] = service
			}
		}
	}

	return project, nil
}

// parseBuildDependsOnFromYAML parses build.depends_on directly from YAML content
func parseBuildDependsOnFromYAML(yamlContent []byte) (map[string][]string, error) {
	var config struct {
		Services map[string]struct {
			Build struct {
				DependsOn interface{} `yaml:"depends_on"`
			} `yaml:"build"`
		} `yaml:"services"`
	}

	err := yaml.Unmarshal(yamlContent, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	result := make(map[string][]string)
	for serviceName, service := range config.Services {
		if service.Build.DependsOn == nil {
			continue
		}

		var deps []string
		switch dependsOn := service.Build.DependsOn.(type) {
		case []interface{}:
			for _, dep := range dependsOn {
				if depStr, ok := dep.(string); ok {
					deps = append(deps, depStr)
				}
			}
		case []string:
			deps = dependsOn
		case string:
			deps = []string{dependsOn}
		default:
			// Try to decode using mapstructure for more complex cases
			if err := mapstructure.Decode(dependsOn, &deps); err != nil {
				continue
			}
		}

		if len(deps) > 0 {
			result[serviceName] = deps
		}
	}

	return result, nil
}