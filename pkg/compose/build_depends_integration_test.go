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

func TestBuildWithDependsOn(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDockerCli := mocks.NewMockCli(ctrl)

	// Create a simple project with build dependencies
	project := &types.Project{
		Name: "test-project",
		Services: types.Services{
			"frontend": {
				Name: "frontend",
				Build: &types.BuildConfig{
					Context: "./frontend",
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

	cs := &composeService{
		dockerCli: mockDockerCli,
	}

	// Test extracting build dependencies
	deps := getBuildDependsOn(project.Services["frontend"])
	assert.DeepEqual(t, deps, []string{"database"})

	// Test validation that missing dependencies are caught
	project.Services["frontend"] = types.ServiceConfig{
		Name: "frontend",
		Build: &types.BuildConfig{
			Context: "./frontend",
			Extensions: types.Extensions{
				"depends_on": []string{"nonexistent"},
			},
		},
	}

	_, err := cs.startBuildDependencies(ctx, project, "frontend")
	assert.ErrorContains(t, err, `depends on "nonexistent" for build, but "nonexistent" is not defined`)
}