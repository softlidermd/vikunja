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

	"github.com/stretchr/testify/require"
	"xorm.io/xorm"
	"xorm.io/xorm/schemas"
)

// Old shapes: neither table has the new column yet.
type tasksBefore20261111120100 struct {
	ID        int64  `xorm:"bigint autoincr not null unique pk"`
	Title     string `xorm:"TEXT not null"`
	ProjectID int64  `xorm:"bigint INDEX not null unique(tasks_project_index)"`
	Index     int64  `xorm:"bigint not null default 0 unique(tasks_project_index)"`
	Done      bool   `xorm:"INDEX null"`
}

func (tasksBefore20261111120100) TableName() string {
	return "tasks"
}

type savedFiltersBefore20261111120100 struct {
	ID      int64  `xorm:"bigint autoincr not null unique pk"`
	OwnerID int64  `xorm:"bigint not null INDEX"`
	Title   string `xorm:"varchar(250) not null"`
}

func (savedFiltersBefore20261111120100) TableName() string {
	return "saved_filters"
}

type tasksAfter20261111120100 struct {
	ID    int64  `xorm:"bigint autoincr not null unique pk"`
	Title string `xorm:"TEXT not null"`
	Size  int64  `xorm:"bigint not null default 0"`
}

func (tasksAfter20261111120100) TableName() string {
	return "tasks"
}

type savedFiltersAfter20261111120100 struct {
	ID          int64  `xorm:"bigint autoincr not null unique pk"`
	Title       string `xorm:"varchar(250) not null"`
	AllProjects bool   `xorm:"default false"`
}

func (savedFiltersAfter20261111120100) TableName() string {
	return "saved_filters"
}

func tablesByName20261111120100(t *testing.T, x *xorm.Engine) map[string]*schemas.Table {
	t.Helper()
	tables, err := x.DBMetas()
	require.NoError(t, err)
	byName := map[string]*schemas.Table{}
	for _, table := range tables {
		byName[table.Name] = table
	}
	return byName
}

func TestSizeAndAllProjectsColumns20261111120100(t *testing.T) {
	x, err := db.CreateTestEngine()
	require.NoError(t, err)

	tables := []any{tasksBefore20261111120100{}, savedFiltersBefore20261111120100{}}
	// x is the process-global test engine; these old-shape tables would leak into other tests.
	t.Cleanup(func() {
		require.NoError(t, x.DropTables(tables...))
	})
	require.NoError(t, x.DropTables(tables...))
	require.NoError(t, x.Sync2(tables...))

	_, err = x.Insert(
		&tasksBefore20261111120100{ID: 1, Title: "existing task", ProjectID: 1, Index: 1},
		&savedFiltersBefore20261111120100{ID: 1, OwnerID: 1, Title: "existing filter"},
	)
	require.NoError(t, err)

	before := tablesByName20261111120100(t, x)
	require.Nil(t, before["tasks"].GetColumn("size"))
	require.Nil(t, before["saved_filters"].GetColumn("all_projects"))
	require.NotEmpty(t, before["tasks"].Indexes)
	require.NotEmpty(t, before["saved_filters"].Indexes)

	require.NoError(t, addSizeAndAllProjects20261111120100(x))

	after := tablesByName20261111120100(t, x)
	require.NotNil(t, after["tasks"].GetColumn("size"))
	require.NotNil(t, after["saved_filters"].GetColumn("all_projects"))
	for _, name := range []string{"tasks", "saved_filters"} {
		for _, column := range before[name].ColumnsSeq() {
			require.NotNilf(t, after[name].GetColumn(column), "migration dropped column %s.%s", name, column)
		}
		for indexName, index := range before[name].Indexes {
			preserved, found := after[name].Indexes[indexName]
			require.Truef(t, found, "migration dropped index %s on %s", indexName, name)
			require.Equal(t, index.Type, preserved.Type)
			require.Equal(t, index.Cols, preserved.Cols)
		}
	}

	task := &tasksAfter20261111120100{}
	found, err := x.ID(1).Get(task)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "existing task", task.Title)
	require.Equal(t, int64(0), task.Size)

	filter := &savedFiltersAfter20261111120100{}
	found, err = x.ID(1).Get(filter)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "existing filter", filter.Title)
	require.False(t, filter.AllProjects)
}
