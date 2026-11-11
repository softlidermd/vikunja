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

package models

import (
	"testing"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/events"
	"code.vikunja.io/api/pkg/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"xorm.io/xorm"
)

// User 2 is promoted to instance admin. Project 1 (owner user 1) is not shared with
// user 2; projects 21 and 22 are archived and hold the undone tasks 35 and 36.

func setUser2Admin(t *testing.T, s *xorm.Session, isAdmin bool) {
	_, err := s.ID(int64(2)).Cols("is_admin").Update(&user.User{IsAdmin: isAdmin})
	require.NoError(t, err)
}

func createAllProjectsFilter(t *testing.T, s *xorm.Session, u *user.User, filter string, allProjects bool) *SavedFilter {
	sf := &SavedFilter{
		Title:       "all projects",
		Filters:     &TaskCollection{Filter: filter},
		AllProjects: allProjects,
	}
	require.NoError(t, sf.Create(s, u))
	return sf
}

func filterViewOfKind(t *testing.T, s *xorm.Session, sf *SavedFilter, kind ProjectViewKind) *ProjectView {
	view := &ProjectView{}
	has, err := s.Where("project_id = ? AND view_kind = ?", getProjectIDFromSavedFilterID(sf.ID), kind).Get(view)
	require.NoError(t, err)
	require.True(t, has)
	return view
}

func readFilterTasks(t *testing.T, s *xorm.Session, u *user.User, sf *SavedFilter) []*Task {
	view := filterViewOfKind(t, s, sf, ProjectViewKindList)
	tc := &TaskCollection{ProjectID: view.ProjectID, ProjectViewID: view.ID}
	res, _, _, err := tc.ReadAll(s, u, "", -1, 0)
	require.NoError(t, err)
	return res.([]*Task)
}

func TestSavedFilterAllProjects_AdminSeesUnsharedProjects(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()
	setUser2Admin(t, s, true)
	admin := &user.User{ID: 2}

	flagged := createAllProjectsFilter(t, s, admin, "done = false", true)
	plain := createAllProjectsFilter(t, s, admin, "done = false", false)

	ids := taskIDsOf(readFilterTasks(t, s, admin, flagged))
	assert.Contains(t, ids, int64(1))
	assert.NotContains(t, ids, int64(2), "done task")
	assert.NotContains(t, ids, int64(35), "archived project 21")
	assert.NotContains(t, ids, int64(36), "archived project 22")

	assert.NotContains(t, taskIDsOf(readFilterTasks(t, s, admin, plain)), int64(1))

	// The owner's admin flag is re-read on every evaluation.
	setUser2Admin(t, s, false)
	assert.NotContains(t, taskIDsOf(readFilterTasks(t, s, admin, flagged)), int64(1))
}

func TestSavedFilterAllProjects_ExistingSyntax(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()
	setUser2Admin(t, s, true)
	admin := &user.User{ID: 2}

	sf := createAllProjectsFilter(t, s, admin, "priority >= 4 && done = false", true)
	tasks := readFilterTasks(t, s, admin, sf)
	assert.Contains(t, taskIDsOf(tasks), int64(3))
	assert.NotContains(t, taskIDsOf(tasks), int64(4), "priority 1")
	for _, task := range tasks {
		assert.GreaterOrEqual(t, int64(task.Priority), int64(4))
		assert.False(t, task.Done)
	}

	// Status mirroring: project 1's testbucket2 holds tasks 3-5.
	board := createAllProjectsFilter(t, s, admin, "project = 1 && done = false", true)
	view := filterViewOfKind(t, s, board, ProjectViewKindKanban)
	colID := addMirrorColumn(t, s, admin, view, "testbucket2")
	buckets := readMirrorBoard(t, s, admin, view)
	assert.ElementsMatch(t, []int64{3, 4, 5}, mirrorTaskIDs(buckets[colID]))
}

func TestSavedFilterAllProjects_NonAdminFlagDropped(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()
	nonAdmin := &user.User{ID: 3}

	sf := createAllProjectsFilter(t, s, nonAdmin, "done = false", true)
	assert.False(t, sf.AllProjects)
	stored, err := GetSavedFilterSimpleByID(s, sf.ID)
	require.NoError(t, err)
	assert.False(t, stored.AllProjects, "create")

	sf.AllProjects = true
	require.NoError(t, sf.Update(s, nonAdmin))
	stored, err = GetSavedFilterSimpleByID(s, sf.ID)
	require.NoError(t, err)
	assert.False(t, stored.AllProjects, "update")

	// A flag that reached the database anyway is ignored for a non-admin owner.
	_, err = s.ID(sf.ID).Cols("all_projects").Update(&SavedFilter{AllProjects: true})
	require.NoError(t, err)
	assert.NotContains(t, taskIDsOf(readFilterTasks(t, s, nonAdmin, sf)), int64(1))
}

func TestSavedFilterAllProjects_RecalcKeepsUnsharedPositions(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()
	setUser2Admin(t, s, true)
	admin := &user.User{ID: 2}

	sf := createAllProjectsFilter(t, s, admin, "done = false", true)
	require.NoError(t, sf.Update(s, admin))

	view := filterViewOfKind(t, s, sf, ProjectViewKindKanban)
	tp := &TaskPosition{}
	has, err := s.Where("project_view_id = ? AND task_id = ?", view.ID, 1).Get(tp)
	require.NoError(t, err)
	require.True(t, has)
	assert.NotZero(t, tp.Position)
	has, err = s.Where("project_view_id = ? AND task_id = ?", view.ID, 1).Exist(&TaskBucket{})
	require.NoError(t, err)
	assert.True(t, has)
}

func TestSavedFilterAllProjects_ListenerAddsUnsharedTasks(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	setUser2Admin(t, s, true)
	adminView, _ := createKanbanFilterView(t, s, 9999, 9999, 2, "done = false")
	nonAdminView, _ := createKanbanFilterView(t, s, 9998, 9998, 3, "done = false")
	_, err := s.In("id", 9999, 9998).Cols("all_projects").Update(&SavedFilter{AllProjects: true})
	require.NoError(t, err)
	require.NoError(t, s.Commit())
	_ = s.Close()

	events.TestListener(t, &TaskUpdatedEvent{
		Task: &Task{ID: 1, ProjectID: 1},
		Doer: &user.User{ID: 1},
	}, &UpdateTaskInSavedFilterViews{})

	for _, table := range []string{"task_buckets", "task_positions"} {
		db.AssertExists(t, table, map[string]interface{}{"task_id": 1, "project_view_id": adminView.ID}, false)
		db.AssertMissing(t, table, map[string]interface{}{"task_id": 1, "project_view_id": nonAdminView.ID})
	}
}

func TestSavedFilterAllProjects_ProjectReadAll(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	readProjects := func(t *testing.T, u *user.User, allProjects bool) map[int64]*Project {
		p := &Project{AllProjects: allProjects, Expand: ProjectExpandableRights}
		res, _, _, err := p.ReadAll(s, u, "", -1, 0)
		require.NoError(t, err)
		byID := map[int64]*Project{}
		for _, project := range res.([]*Project) {
			byID[project.ID] = project
		}
		return byID
	}

	t.Run("non-admin flag is ignored", func(t *testing.T) {
		nonAdmin := &user.User{ID: 2}
		withFlag := readProjects(t, nonAdmin, true)
		assert.NotContains(t, withFlag, int64(1))
		assert.ElementsMatch(t, keysOf(readProjects(t, nonAdmin, false)), keysOf(withFlag))
	})

	t.Run("admin gets every non-archived project", func(t *testing.T) {
		setUser2Admin(t, s, true)
		admin := &user.User{ID: 2}
		assert.NotContains(t, readProjects(t, admin, false), int64(1))

		all := readProjects(t, admin, true)
		require.Contains(t, all, int64(1))
		require.NotNil(t, all[1].MaxPermission)
		assert.Equal(t, PermissionAdmin, *all[1].MaxPermission)
		assert.NotContains(t, all, int64(21))
		assert.NotContains(t, all, int64(22))
	})
}

func keysOf(m map[int64]*Project) []int64 {
	keys := make([]int64, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
