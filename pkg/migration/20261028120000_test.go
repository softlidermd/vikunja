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

package migration

import (
	"testing"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"xorm.io/builder"
)

func TestUnsetNullPriorities20261028120000(t *testing.T) {
	x, err := db.CreateTestEngine()
	require.NoError(t, err)
	// The full model struct: a partial Sync2 on the shared engine would drop the table's indexes.
	require.NoError(t, x.Sync2(&models.Task{}))

	const title = "null priority test 20261028120000"
	t.Cleanup(func() {
		_, err := x.Where(builder.Eq{"title": title}).Delete(&models.Task{})
		require.NoError(t, err)
	})

	// Indexes far above the fixtures: (project_id, index) is unique.
	nullTask := &models.Task{Title: title, ProjectID: 1, Index: 2026102801}
	highTask := &models.Task{Title: title, ProjectID: 1, Index: 2026102802, Priority: 3}
	_, err = x.Insert(nullTask, highTask)
	require.NoError(t, err)
	_, err = x.Exec("UPDATE tasks SET priority = NULL WHERE id = ?", nullTask.ID)
	require.NoError(t, err)

	require.NoError(t, unsetNullPriorities20261028120000(x))

	priorities := map[int64]int64{}
	rows, err := x.QueryInterface("SELECT id, priority FROM tasks WHERE title = ? AND priority IS NOT NULL", title)
	require.NoError(t, err)
	for _, row := range rows {
		priorities[row["id"].(int64)] = row["priority"].(int64)
	}
	assert.Equal(t, map[int64]int64{nullTask.ID: 0, highTask.ID: 3}, priorities)
}
