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
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"gotest.tools/v3/assert"
)

func TestEnhanceProjectWithBuildDependsOn(t *testing.T) {
	// Create a temporary compose file with native build.depends_on syntax
	tmpDir := t.TempDir()
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	
	composeContent := `
name: test-enhance
services:
  frontend:
    build:
      context: ./frontend
      depends_on:
        - database
        - cache
    depends_on:
      - database

  backend:
    build:
      context: ./backend
      depends_on: database

  database:
    image: postgres:13

  cache:
    image: redis:6
`

	err := os.WriteFile(composeFile, []byte(composeContent), 0644)
	assert.NilError(t, err)

	// Create a project with the compose file
	project := &types.Project{
		Name:         "test-enhance",
		ComposeFiles: []string{composeFile},
		Services: types.Services{
			"frontend": {
				Name: "frontend",
				Build: &types.BuildConfig{
					Context: "./frontend",
				},
			},
			"backend": {
				Name: "backend",
				Build: &types.BuildConfig{
					Context: "./backend",
				},
			},
			"database": {
				Name:  "database",
				Image: "postgres:13",
			},
			"cache": {
				Name:  "cache",
				Image: "redis:6",
			},
		},
	}

	// Enhance the project
	enhanced, err := EnhanceProjectWithBuildDependsOn(project)
	assert.NilError(t, err)

	// Check that build dependencies were extracted and injected
	frontendDeps := getBuildDependsOnEnhanced(enhanced.Services["frontend"])
	expectedFrontendDeps := []string{"database", "cache"}
	assert.DeepEqual(t, frontendDeps, expectedFrontendDeps)

	backendDeps := getBuildDependsOnEnhanced(enhanced.Services["backend"])
	expectedBackendDeps := []string{"database"}
	assert.DeepEqual(t, backendDeps, expectedBackendDeps)

	// Services without build dependencies should have none
	databaseDeps := getBuildDependsOnEnhanced(enhanced.Services["database"])
	assert.Equal(t, len(databaseDeps), 0)
}

func TestExtractBuildDependsOnFromFile(t *testing.T) {
	// Create a temporary compose file
	tmpDir := t.TempDir()
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	
	composeContent := `
name: test-extract
services:
  app1:
    build:
      context: ./app1
      depends_on:
        - db
        - cache
  
  app2:
    build:
      context: ./app2
      depends_on: db

  db:
    image: postgres:13
  
  cache:
    image: redis:6
`

	err := os.WriteFile(composeFile, []byte(composeContent), 0644)
	assert.NilError(t, err)

	// Extract dependencies
	deps, err := extractBuildDependsOnFromFile(composeFile)
	assert.NilError(t, err)

	expectedApp1Deps := []string{"db", "cache"}
	expectedApp2Deps := []string{"db"}

	assert.DeepEqual(t, deps["app1"], expectedApp1Deps)
	assert.DeepEqual(t, deps["app2"], expectedApp2Deps)
	assert.Equal(t, len(deps), 2) // Only app1 and app2 should have build deps
}