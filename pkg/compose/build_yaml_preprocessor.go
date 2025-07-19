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
	"path/filepath"

	"github.com/compose-spec/compose-go/v2/types"
	"gopkg.in/yaml.v3"
)

// BuildDependsOnData stores extracted build.depends_on information
type BuildDependsOnData struct {
	FilePath string
	Services map[string][]string
}

// PreprocessComposeBuildDependsOn removes build.depends_on from compose files to avoid schema validation errors
// Returns the extracted build.depends_on data and paths to temporary files without build.depends_on
func PreprocessComposeBuildDependsOn(configPaths []string) ([]BuildDependsOnData, []string, error) {
	var extractedData []BuildDependsOnData
	var tempFiles []string

	for _, configPath := range configPaths {
		data, tempFile, err := preprocessSingleFile(configPath)
		if err != nil {
			// Cleanup temp files on error
			for _, tf := range tempFiles {
				os.Remove(tf)
			}
			return nil, nil, err
		}
		
		if len(data.Services) > 0 {
			extractedData = append(extractedData, data)
		}
		tempFiles = append(tempFiles, tempFile)
	}

	return extractedData, tempFiles, nil
}

// preprocessSingleFile processes a single compose file
func preprocessSingleFile(filePath string) (BuildDependsOnData, string, error) {
	data := BuildDependsOnData{
		FilePath: filePath,
		Services: make(map[string][]string),
	}

	// Read the original file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return data, "", err
	}

	// Parse YAML to extract build.depends_on
	var config map[string]interface{}
	if err := yaml.Unmarshal(content, &config); err != nil {
		// If we can't parse the YAML, just use the original file
		return data, filePath, nil
	}

	// Extract build.depends_on and remove it from the config
	var modified bool
	if services, ok := config["services"].(map[string]interface{}); ok {
		for serviceName, serviceConfig := range services {
			if serviceMap, ok := serviceConfig.(map[string]interface{}); ok {
				if buildConfig, ok := serviceMap["build"].(map[string]interface{}); ok {
					if dependsOn, exists := buildConfig["depends_on"]; exists {
						// Extract the depends_on data
						if deps := extractDependsOnList(dependsOn); len(deps) > 0 {
							data.Services[serviceName] = deps
						}
						
						// Remove depends_on from the build config
						delete(buildConfig, "depends_on")
						modified = true
					}
				}
			}
		}
	}

	// If no build.depends_on was found, just return the original file path
	if !modified {
		return data, filePath, nil
	}

	// Create a temporary file with the modified content
	tempFile, err := createTempFileWithModifiedYAML(filePath, config)
	if err != nil {
		return data, "", err
	}

	return data, tempFile, nil
}

// extractDependsOnList converts various depends_on formats to a string slice
func extractDependsOnList(dependsOn interface{}) []string {
	switch v := dependsOn.(type) {
	case []interface{}:
		var deps []string
		for _, dep := range v {
			if depStr, ok := dep.(string); ok {
				deps = append(deps, depStr)
			}
		}
		return deps
	case []string:
		return v
	case string:
		return []string{v}
	default:
		return nil
	}
}

// createTempFileWithModifiedYAML creates a temporary file with the modified YAML content
func createTempFileWithModifiedYAML(originalPath string, config map[string]interface{}) (string, error) {
	// Marshal the modified config back to YAML
	modifiedContent, err := yaml.Marshal(config)
	if err != nil {
		return "", err
	}

	// Create a temporary file
	ext := filepath.Ext(originalPath)
	if ext == "" {
		ext = ".yml"
	}
	
	tempFile, err := os.CreateTemp("", "compose-build-deps-*"+ext)
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	// Write the modified content
	if _, err := tempFile.Write(modifiedContent); err != nil {
		os.Remove(tempFile.Name())
		return "", err
	}

	return tempFile.Name(), nil
}

// CleanupTempFiles removes temporary files created during preprocessing
func CleanupTempFiles(tempFiles []string) {
	for _, tempFile := range tempFiles {
		os.Remove(tempFile)
	}
}

// RestoreBuildDependsOn adds the extracted build.depends_on data back to the project as extensions
func RestoreBuildDependsOn(project *types.Project, extractedData []BuildDependsOnData) *types.Project {
	for _, data := range extractedData {
		for serviceName, deps := range data.Services {
			if service, exists := project.Services[serviceName]; exists && service.Build != nil {
				if service.Build.Extensions == nil {
					service.Build.Extensions = make(types.Extensions)
				}
				service.Build.Extensions["build-depends-on"] = deps
				project.Services[serviceName] = service
			}
		}
	}
	return project
}