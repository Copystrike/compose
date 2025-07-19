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

	"github.com/compose-spec/compose-go/v2/types"
	"gotest.tools/v3/assert"
)

func Test_getBuildDependsOn(t *testing.T) {
	tests := []struct {
		name     string
		service  types.ServiceConfig
		expected []string
	}{
		{
			name: "no build config",
			service: types.ServiceConfig{
				Build: nil,
			},
			expected: nil,
		},
		{
			name: "build config without depends_on",
			service: types.ServiceConfig{
				Build: &types.BuildConfig{
					Context: ".",
				},
			},
			expected: nil,
		},
		{
			name: "build config with depends_on as string slice",
			service: types.ServiceConfig{
				Build: &types.BuildConfig{
					Context: ".",
					Extensions: types.Extensions{
						"depends_on": []string{"database", "cache"},
					},
				},
			},
			expected: []string{"database", "cache"},
		},
		{
			name: "build config with depends_on as interface slice",
			service: types.ServiceConfig{
				Build: &types.BuildConfig{
					Context: ".",
					Extensions: types.Extensions{
						"depends_on": []interface{}{"database", "cache"},
					},
				},
			},
			expected: []string{"database", "cache"},
		},
		{
			name: "build config with depends_on as single string",
			service: types.ServiceConfig{
				Build: &types.BuildConfig{
					Context: ".",
					Extensions: types.Extensions{
						"depends_on": "database",
					},
				},
			},
			expected: []string{"database"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getBuildDependsOn(tt.service)
			assert.DeepEqual(t, result, tt.expected)
		})
	}
}