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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreprocessComposeBuildDependsOn(t *testing.T) {
	tests := []struct {
		name           string
		yamlContent    string
		expectedData   map[string][]string
		shouldModify   bool
	}{
		{
			name: "simple build depends_on",
			yamlContent: `
services:
  frontend:
    build:
      context: .
      depends_on:
        - backend
        - database
  backend:
    image: nginx`,
			expectedData: map[string][]string{
				"frontend": {"backend", "database"},
			},
			shouldModify: true,
		},
		{
			name: "string build depends_on",
			yamlContent: `
services:
  app:
    build:
      context: .
      depends_on: database
  database:
    image: postgres`,
			expectedData: map[string][]string{
				"app": {"database"},
			},
			shouldModify: true,
		},
		{
			name: "no build depends_on",
			yamlContent: `
services:
  app:
    build:
      context: .
  database:
    image: postgres`,
			expectedData:   map[string][]string{},
			shouldModify: false,
		},
		{
			name: "mixed services",
			yamlContent: `
services:
  frontend:
    build:
      context: ./frontend
      depends_on: [backend]
  backend:
    build:
      context: ./backend
  database:
    image: postgres`,
			expectedData: map[string][]string{
				"frontend": {"backend"},
			},
			shouldModify: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file with the test YAML content
			tempDir := t.TempDir()
			yamlFile := filepath.Join(tempDir, "docker-compose.yml")
			require.NoError(t, os.WriteFile(yamlFile, []byte(tt.yamlContent), 0644))

			// Test preprocessing
			extractedData, tempFiles, err := PreprocessComposeBuildDependsOn([]string{yamlFile})
			require.NoError(t, err)
			defer CleanupTempFiles(tempFiles)

			// Verify extracted data
			if len(tt.expectedData) == 0 {
				assert.Empty(t, extractedData)
			} else {
				require.Len(t, extractedData, 1)
				assert.Equal(t, yamlFile, extractedData[0].FilePath)
				assert.Equal(t, tt.expectedData, extractedData[0].Services)
			}

			// Verify temp files
			require.Len(t, tempFiles, 1)
			if tt.shouldModify {
				// Should create a temp file
				assert.NotEqual(t, yamlFile, tempFiles[0])
				
				// Verify the temp file doesn't contain build.depends_on
				tempContent, err := os.ReadFile(tempFiles[0])
				require.NoError(t, err)
				assert.NotContains(t, string(tempContent), "depends_on:")
			} else {
				// Should use the original file
				assert.Equal(t, yamlFile, tempFiles[0])
			}
		})
	}
}

func TestExtractDependsOnList(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected []string
	}{
		{
			name:     "string slice",
			input:    []string{"service1", "service2"},
			expected: []string{"service1", "service2"},
		},
		{
			name:     "interface slice",
			input:    []interface{}{"service1", "service2"},
			expected: []string{"service1", "service2"},
		},
		{
			name:     "single string",
			input:    "service1",
			expected: []string{"service1"},
		},
		{
			name:     "invalid type",
			input:    123,
			expected: nil,
		},
		{
			name:     "mixed interface slice",
			input:    []interface{}{"service1", 123, "service2"},
			expected: []string{"service1", "service2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractDependsOnList(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}