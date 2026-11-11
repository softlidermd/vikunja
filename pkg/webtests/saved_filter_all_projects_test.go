// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.


package webtests

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Project 1 belongs to user 1 and is not shared with user 2.
func TestSavedFilterAllProjects_ProjectsEndpoint(t *testing.T) {
	e, err := setupTestEnv()
	require.NoError(t, err)

	maxPermissionOfProject1 := func(t *testing.T) (found bool, maxPermission int) {
		rec := humaRequest(t, e, http.MethodGet, "/api/v2/projects?all_projects=true&expand=permissions&s=1", "", humaTokenFor(t, &testuser2), "")
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
		var body struct {
			Items []struct {
				ID            int64 `json:"id"`
				MaxPermission *int  `json:"max_permission"`
			} `json:"items"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		for _, p := range body.Items {
			if p.ID == 1 {
				require.NotNil(t, p.MaxPermission)
				return true, *p.MaxPermission
			}
		}
		return false, 0
	}

	found, _ := maxPermissionOfProject1(t)
	assert.False(t, found, "non-admin must not get unshared projects")

	promoteToAdmin(t, testuser2.ID)
	found, maxPermission := maxPermissionOfProject1(t)
	assert.True(t, found, "admin gets every project")
	assert.Equal(t, 2, maxPermission)
}
