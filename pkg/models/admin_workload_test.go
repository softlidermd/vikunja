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
	"time"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Fixtures already hold open work for users 1 and 2, so every assertion is a delta or an id check.
func TestWorkload_GetWorkload(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()
	u := &user.User{ID: 1}

	person := func(w *Workload, userID int64) *WorkloadPerson {
		for _, p := range w.People {
			if p.User.ID == userID {
				return p
			}
		}
		return &WorkloadPerson{}
	}
	delta := func(after, before WorkloadTotals) WorkloadTotals {
		return WorkloadTotals{
			Hours:       after.Hours - before.Hours,
			Unestimated: after.Unestimated - before.Unestimated,
			Long:        after.Long - before.Long,
			TaskCount:   after.TaskCount - before.TaskCount,
			Ideas:       after.Ideas - before.Ideas,
		}
	}
	taskIDs := func(tasks []*WorkloadTask) []int64 {
		ids := []int64{}
		for _, t := range tasks {
			ids = append(ids, t.ID)
		}
		return ids
	}
	assign := func(taskID int64, userIDs ...int64) {
		for _, id := range userIDs {
			_, err := s.Insert(&TaskAssginee{TaskID: taskID, UserID: id})
			require.NoError(t, err)
		}
	}

	before, err := GetWorkload(s)
	require.NoError(t, err)

	// Project P: a fresh Idea board; planned tasks go to To-Do explicitly.
	p, _, buckets := newIdeaBoard(t, s, u)
	todo := buckets[1].ID
	shared := newBoardTask(t, s, u, p.ID, &Task{BucketID: todo, Size: 8})
	assign(shared, 1, 2)
	long := newBoardTask(t, s, u, p.ID, &Task{BucketID: todo, Size: TaskSizeLong, DueDate: time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)})
	assign(long, 1)
	unestimated := newBoardTask(t, s, u, p.ID, &Task{BucketID: todo, Priority: 3})
	assign(unestimated, 1)
	idea := newBoardTask(t, s, u, p.ID, &Task{Size: 40})
	assign(idea, 1)
	unassigned := newBoardTask(t, s, u, p.ID, &Task{BucketID: todo, Size: 4})
	unassignedIdea := newBoardTask(t, s, u, p.ID, &Task{})
	done := newBoardTask(t, s, u, p.ID, &Task{Done: true, Size: 8})
	assign(done, 1)
	deleted := newBoardTask(t, s, u, p.ID, &Task{BucketID: todo, Size: 8})
	assign(deleted, 1)
	require.NoError(t, (&Task{ID: deleted}).Delete(s, u))

	// Project Q: archived.
	q, _, qBuckets := newIdeaBoard(t, s, u)
	archived := newBoardTask(t, s, u, q.ID, &Task{BucketID: qBuckets[1].ID, Size: 8})
	assign(archived, 1)
	_, err = s.ID(q.ID).Cols("is_archived").Update(&Project{IsArchived: true})
	require.NoError(t, err)

	// Project R: two manual boards, so no status view; its task counts as planned although it sits in Idea.
	r, _, _ := newIdeaBoard(t, s, u)
	require.NoError(t, (&ProjectView{
		ProjectID:               r.ID,
		Title:                   "second board",
		ViewKind:                ProjectViewKindKanban,
		BucketConfigurationMode: BucketConfigurationModeManual,
	}).Create(s, u))
	noStatus := newBoardTask(t, s, u, r.ID, &Task{})
	assign(noStatus, 1)

	after, err := GetWorkload(s)
	require.NoError(t, err)

	user1 := person(after, 1)
	assert.Equal(t, WorkloadTotals{Hours: 8, Unestimated: 2, Long: 1, TaskCount: 4, Ideas: 1}, delta(user1.WorkloadTotals, person(before, 1).WorkloadTotals))
	assert.Equal(t, WorkloadTotals{Hours: 8, TaskCount: 1}, delta(person(after, 2).WorkloadTotals, person(before, 2).WorkloadTotals))
	assert.Equal(t, WorkloadTotals{Hours: 4, TaskCount: 1, Ideas: 1}, delta(after.Unassigned.WorkloadTotals, before.Unassigned.WorkloadTotals))
	assert.Equal(t, WorkloadTotals{Hours: 12, Unestimated: 2, Long: 1, TaskCount: 5, Ideas: 2}, delta(after.Total, before.Total))

	// Due date first (unset last), then priority desc, then id.
	ordered := []int64{}
	for _, id := range taskIDs(user1.Tasks) {
		if id == long || id == unestimated || id == shared || id == noStatus {
			ordered = append(ordered, id)
		}
	}
	assert.Equal(t, []int64{long, unestimated, shared, noStatus}, ordered)
	assert.Contains(t, taskIDs(person(after, 2).Tasks), shared)
	assert.Contains(t, taskIDs(after.Unassigned.Tasks), unassigned)

	listed := taskIDs(after.Unassigned.Tasks)
	for _, block := range after.People {
		listed = append(listed, taskIDs(block.Tasks)...)
	}
	for _, id := range []int64{idea, unassignedIdea, done, deleted, archived} {
		assert.NotContains(t, listed, id)
	}

	for _, task := range user1.Tasks {
		switch task.ID {
		case shared:
			assert.Equal(t, "To-Do", task.Status)
			assert.Equal(t, p.Title, task.ProjectTitle)
		case noStatus:
			assert.Empty(t, task.Status)
		}
	}
}
