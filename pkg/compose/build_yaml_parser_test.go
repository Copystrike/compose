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
	"testing"

	"gotest.tools/v3/assert"
)

func TestParseBuildDependsOnFromYAML(t *testing.T) {
	yamlContent := `
name: test-build-depends-on
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

	buildDeps, err := parseBuildDependsOnFromYAML([]byte(yamlContent))
	assert.NilError(t, err)
	
	expectedFrontend := []string{"database", "cache"}
	expectedBackend := []string{"database"}
	
	assert.DeepEqual(t, buildDeps["frontend"], expectedFrontend)
	assert.DeepEqual(t, buildDeps["backend"], expectedBackend)
	assert.Equal(t, len(buildDeps), 2) // Only frontend and backend should have build deps
}

func TestParseBuildDependsOnFromYAMLEmpty(t *testing.T) {
	yamlContent := `
name: test-no-build-deps
services:
  app:
    build:
      context: ./app

  database:
    image: postgres:13
`

	buildDeps, err := parseBuildDependsOnFromYAML([]byte(yamlContent))
	assert.NilError(t, err)
	assert.Equal(t, len(buildDeps), 0)
}