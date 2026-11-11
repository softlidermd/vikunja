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
	"code.vikunja.io/api/pkg/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"xorm.io/xorm"
)

// newIdeaBoard creates a project and returns its Kanban view and buckets: Idea (default), To-Do, Doing, Done.
func newIdeaBoard(t *testing.T, s *xorm.Session, u *user.User) (*Project, *ProjectView, []*Bucket) {
	project := &Project{Title: "idea board"}
	require.NoError(t, project.Create(s, u))

	view := &ProjectView{}
	has, err := s.Where("project_id = ? AND view_kind = ?", project.ID, ProjectViewKindKanban).Get(view)
	require.NoError(t, err)
	require.True(t, has)

	buckets := []*Bucket{}
	require.NoError(t, s.Where("project_view_id = ?", view.ID).OrderBy("position asc, id asc").Find(&buckets))
	require.Len(t, buckets, 4)
	require.Equal(t, "Idea", buckets[0].Title)
	require.Equal(t, buckets[0].ID, view.DefaultBucketID)
	return project, view, buckets
}

func newBoardTask(t *testing.T, s *xorm.Session, u *user.User, projectID int64, task *Task) int64 {
	task.Title = "planned"
	task.ProjectID = projectID
	require.NoError(t, task.Create(s, u))
	return task.ID
}

func moveBoardTask(t *testing.T, s *xorm.Session, u *user.User, project *Project, view *ProjectView, taskID, bucketID int64) *TaskBucket {
	tb := &TaskBucket{TaskID: taskID, BucketID: bucketID, ProjectViewID: view.ID, ProjectID: project.ID}
	require.NoError(t, tb.Update(s, u))
	return tb
}

func TestTaskCreate_LandsInIdea(t *testing.T) {
	u := &user.User{ID: 1}

	t.Run("bucketless create", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()
		project, view, buckets := newIdeaBoard(t, s, u)

		id := newBoardTask(t, s, u, project.ID, &Task{})
		assert.Equal(t, buckets[0].ID, mirrorRow(t, s, view.ID, id))
	})
	t.Run("explicit bucket is kept", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()
		project, view, buckets := newIdeaBoard(t, s, u)

		id := newBoardTask(t, s, u, project.ID, &Task{BucketID: buckets[2].ID})
		assert.Equal(t, buckets[2].ID, mirrorRow(t, s, view.ID, id))
	})
}

func TestPlannedBucket(t *testing.T) {
	u := &user.User{ID: 1}
	const idea, todo, doing, done = 0, 1, 2, 3

	t.Run("repeat via Task.Update", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()
		project, view, buckets := newIdeaBoard(t, s, u)
		id := newBoardTask(t, s, u, project.ID, &Task{RepeatAfter: 3600})

		task := &Task{ID: id, Done: true, RepeatAfter: 3600}
		require.NoError(t, task.Update(s, u))
		assert.False(t, task.Done)
		assert.Equal(t, buckets[todo].ID, mirrorRow(t, s, view.ID, id))
	})
	t.Run("repeat via drop on Done", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()
		project, view, buckets := newIdeaBoard(t, s, u)
		id := newBoardTask(t, s, u, project.ID, &Task{RepeatAfter: 3600})

		tb := moveBoardTask(t, s, u, project, view, id, buckets[done].ID)
		assert.False(t, tb.Task.Done)
		assert.Equal(t, buckets[todo].ID, mirrorRow(t, s, view.ID, id))
	})
	t.Run("reopen of a done task", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()
		project, view, buckets := newIdeaBoard(t, s, u)
		id := newBoardTask(t, s, u, project.ID, &Task{Done: true})
		require.Equal(t, buckets[done].ID, mirrorRow(t, s, view.ID, id))

		require.NoError(t, (&Task{ID: id, Done: false}).Update(s, u))
		assert.Equal(t, buckets[todo].ID, mirrorRow(t, s, view.ID, id))
	})
	t.Run("move from another project", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()
		project, view, buckets := newIdeaBoard(t, s, u)
		id := newBoardTask(t, s, u, 1, &Task{})

		require.NoError(t, (&Task{ID: id, ProjectID: project.ID}).Update(s, u))
		assert.Equal(t, buckets[todo].ID, mirrorRow(t, s, view.ID, id))
	})
	t.Run("delete of Doing holding a task", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()
		project, view, buckets := newIdeaBoard(t, s, u)
		id := newBoardTask(t, s, u, project.ID, &Task{})
		moveBoardTask(t, s, u, project, view, id, buckets[doing].ID)

		require.NoError(t, (&Bucket{ID: buckets[doing].ID, ProjectViewID: view.ID, ProjectID: project.ID}).Delete(s, u))
		assert.Equal(t, buckets[todo].ID, mirrorRow(t, s, view.ID, id))
	})
	t.Run("delete of the leftmost bucket without explicit default", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()
		project, view, buckets := newIdeaBoard(t, s, u)
		id := newBoardTask(t, s, u, project.ID, &Task{})
		require.Equal(t, buckets[idea].ID, mirrorRow(t, s, view.ID, id))
		_, err := s.ID(view.ID).Cols("default_bucket_id").Update(&ProjectView{DefaultBucketID: 0})
		require.NoError(t, err)

		require.NoError(t, (&Bucket{ID: buckets[idea].ID, ProjectViewID: view.ID, ProjectID: project.ID}).Delete(s, u))
		assert.Equal(t, buckets[todo].ID, mirrorRow(t, s, view.ID, id))
	})
	t.Run("list view with tasks switched to a manual Kanban", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()
		project, _, _ := newIdeaBoard(t, s, u)
		id := newBoardTask(t, s, u, project.ID, &Task{})

		list := &ProjectView{}
		has, err := s.Where("project_id = ? AND view_kind = ?", project.ID, ProjectViewKindList).Get(list)
		require.NoError(t, err)
		require.True(t, has)
		list.ViewKind = ProjectViewKindKanban
		list.BucketConfigurationMode = BucketConfigurationModeManual
		require.NoError(t, list.Update(s, u))

		buckets := []*Bucket{}
		require.NoError(t, s.Where("project_view_id = ?", list.ID).OrderBy("position asc, id asc").Find(&buckets))
		require.Len(t, buckets, 4)
		assert.Equal(t, buckets[idea].ID, list.DefaultBucketID)
		assert.Equal(t, buckets[todo].ID, mirrorRow(t, s, list.ID, id))
	})
	t.Run("delete of To-Do on an Idea board goes to Doing", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()
		project, view, buckets := newIdeaBoard(t, s, u)
		id := newBoardTask(t, s, u, project.ID, &Task{})
		moveBoardTask(t, s, u, project, view, id, buckets[todo].ID)

		require.NoError(t, (&Bucket{ID: buckets[todo].ID, ProjectViewID: view.ID, ProjectID: project.ID}).Delete(s, u))
		assert.Equal(t, buckets[doing].ID, mirrorRow(t, s, view.ID, id))
	})
}
