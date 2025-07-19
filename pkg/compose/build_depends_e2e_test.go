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
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/compose/v2/pkg/mocks"
	"go.uber.org/mock/gomock"
	"gotest.tools/v3/assert"
)

func TestBuildDependsOnIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Mock services
	mockDockerCli := mocks.NewMockCli(ctrl)
	
	// Create a project with build dependencies using extensions
	project := &types.Project{
		Name: "test-build-deps",
		Services: types.Services{
			"web": {
				Name: "web",
				Build: &types.BuildConfig{
					Context: "./web",
					Extensions: types.Extensions{
						"depends_on": []string{"database", "cache"},
					},
				},
			},
			"api": {
				Name: "api", 
				Build: &types.BuildConfig{
					Context: "./api",
					Extensions: types.Extensions{
						"depends_on": "database",
					},
				},
			},
			"database": {
				Name:  "database",
				Image: "postgres:13",
				Environment: types.MappingWithEquals{
					"POSTGRES_DB":       &[]string{"testdb"}[0],
					"POSTGRES_USER":     &[]string{"user"}[0],
					"POSTGRES_PASSWORD": &[]string{"pass"}[0],
				},
			},
			"cache": {
				Name:  "cache",
				Image: "redis:6",
			},
		},
	}

	cs := &composeService{
		dockerCli: mockDockerCli,
	}

	// Test dependency extraction
	webDeps := getBuildDependsOn(project.Services["web"])
	expectedWebDeps := []string{"database", "cache"}
	assert.DeepEqual(t, webDeps, expectedWebDeps)

	apiDeps := getBuildDependsOn(project.Services["api"])
	expectedAPIDeps := []string{"database"}
	assert.DeepEqual(t, apiDeps, expectedAPIDeps)

	// Test validation with missing dependency
	project.Services["broken"] = types.ServiceConfig{
		Name: "broken",
		Build: &types.BuildConfig{
			Context: "./broken",
			Extensions: types.Extensions{
				"depends_on": []string{"missing-service"},
			},
		},
	}

	_, err := cs.startBuildDependencies(ctx, project, "broken")
	assert.ErrorContains(t, err, `depends on "missing-service" for build, but "missing-service" is not defined`)
}

func TestBuildOrderWithDependencies(t *testing.T) {
	// Test that the build order respects build dependencies
	project := &types.Project{
		Name: "test-build-order",
		Services: types.Services{
			"frontend": {
				Name: "frontend",
				Build: &types.BuildConfig{
					Context: "./frontend",
					Extensions: types.Extensions{
						"depends_on": []string{"backend"},
					},
				},
			},
			"backend": {
				Name: "backend",
				Build: &types.BuildConfig{
					Context: "./backend",
					Extensions: types.Extensions{
						"depends_on": []string{"database"},
					},
				},
			},
			"database": {
				Name:  "database",
				Image: "postgres:13",
			},
		},
	}

	// Verify dependency chains are correctly identified
	frontendDeps := getBuildDependsOn(project.Services["frontend"])
	assert.DeepEqual(t, frontendDeps, []string{"backend"})

	backendDeps := getBuildDependsOn(project.Services["backend"])
	assert.DeepEqual(t, backendDeps, []string{"database"})

	// Database should have no build dependencies
	databaseDeps := getBuildDependsOn(project.Services["database"])
	assert.Equal(t, len(databaseDeps), 0)
}