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
	"context"
	"strconv"
	"testing"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/events"
	"code.vikunja.io/api/pkg/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"xorm.io/xorm"
)

// Fixture project 1 has one manual Kanban view (4): bucket 1 testbucket1 (default), 2 testbucket2
// (limit 3, tasks 3-5), 3 testbucket3 (done, tasks 2 and 6-8).

func createMirrorFilter(t *testing.T, s *xorm.Session, u *user.User, filter string) (view *ProjectView, todoID int64) {
	sf := &SavedFilter{
		Title:   "mirror",
		Filters: &TaskCollection{Filter: filter},
	}
	require.NoError(t, sf.Create(s, u))

	view = &ProjectView{}
	has, err := s.
		Where("project_id = ? AND view_kind = ?", getProjectIDFromSavedFilterID(sf.ID), ProjectViewKindKanban).
		Get(view)
	require.NoError(t, err)
	require.True(t, has)
	return view, view.DefaultBucketID
}

func addMirrorColumn(t *testing.T, s *xorm.Session, u *user.User, view *ProjectView, title string) int64 {
	b := &Bucket{
		ProjectViewID: view.ID,
		Title:         title,
	}
	require.NoError(t, b.Create(s, u))
	return b.ID
}

func readMirrorBoard(t *testing.T, s *xorm.Session, u *user.User, view *ProjectView) map[int64]*Bucket {
	tc := &TaskCollection{
		ProjectID:     view.ProjectID,
		ProjectViewID: view.ID,
	}
	res, _, _, err := tc.ReadAll(s, u, "", 1, 50)
	require.NoError(t, err)
	buckets := map[int64]*Bucket{}
	for _, b := range res.([]*Bucket) {
		buckets[b.ID] = b
	}
	return buckets
}

func mirrorTaskIDs(b *Bucket) []int64 {
	ids := []int64{}
	for _, task := range b.Tasks {
		ids = append(ids, task.ID)
	}
	return ids
}

func mirrorRow(t *testing.T, s *xorm.Session, viewID, taskID int64) int64 {
	tb := &TaskBucket{}
	has, err := s.Where("task_id = ? AND project_view_id = ?", taskID, viewID).Get(tb)
	require.NoError(t, err)
	require.True(t, has)
	return tb.BucketID
}

func setMirrorRow(t *testing.T, s *xorm.Session, viewID, taskID, bucketID int64) {
	require.NoError(t, (&TaskBucket{
		TaskID:        taskID,
		ProjectViewID: viewID,
		BucketID:      bucketID,
	}).upsert(s))
}

func moveMirrorTask(s *xorm.Session, u *user.User, view *ProjectView, taskID, bucketID int64) (*TaskBucket, error) {
	tb := &TaskBucket{
		TaskID:        taskID,
		BucketID:      bucketID,
		ProjectViewID: view.ID,
		ProjectID:     view.ProjectID,
	}
	return tb, tb.Update(s, u)
}

func TestSavedFilterMirror_T1_BoardReadMirrorsStatus(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()
	u := &user.User{ID: 1}

	view, todoID := createMirrorFilter(t, s, u, "project = 1 && done = false")
	colID := addMirrorColumn(t, s, u, view, " TESTBUCKET2 ")

	board := readMirrorBoard(t, s, u, view)
	assert.ElementsMatch(t, []int64{3, 4, 5}, mirrorTaskIDs(board[colID]))
	assert.Equal(t, int64(3), board[colID].Count)
	assert.Contains(t, mirrorTaskIDs(board[todoID]), int64(1))
}

func TestSavedFilterMirror_T2_UnmatchedHandPlacementStays(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()
	u := &user.User{ID: 1}

	view, _ := createMirrorFilter(t, s, u, "project = 1 && done = false")
	addMirrorColumn(t, s, u, view, " TESTBUCKET2 ")
	customID := addMirrorColumn(t, s, u, view, "Custom")
	setMirrorRow(t, s, view.ID, 1, customID)

	readMirrorBoard(t, s, u, view)
	assert.Equal(t, customID, mirrorRow(t, s, view.ID, 1))
}

func TestSavedFilterMirror_T2b_StatusWinsOverHandPlacement(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()
	u := &user.User{ID: 1}

	view, _ := createMirrorFilter(t, s, u, "project = 1 && done = false")
	colID := addMirrorColumn(t, s, u, view, " TESTBUCKET2 ")
	customID := addMirrorColumn(t, s, u, view, "Custom")
	setMirrorRow(t, s, view.ID, 3, customID)

	readMirrorBoard(t, s, u, view)
	assert.Equal(t, colID, mirrorRow(t, s, view.ID, 3))
}

func TestSavedFilterMirror_T3_FlatReadAgreesWithBoard(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()
	u := &user.User{ID: 1}

	view, _ := createMirrorFilter(t, s, u, "project = 1 && done = false")
	colID := addMirrorColumn(t, s, u, view, " TESTBUCKET2 ")
	board := readMirrorBoard(t, s, u, view)

	tc := &TaskCollection{
		ProjectID:     view.ProjectID,
		ProjectViewID: view.ID,
		Filter:        "bucket_id = " + strconv.FormatInt(colID, 10),
	}
	res, _, total, err := tc.ReadAll(s, u, "", 1, 50)
	require.NoError(t, err)
	ids := []int64{}
	for _, task := range res.([]*Task) {
		ids = append(ids, task.ID)
	}
	assert.ElementsMatch(t, []int64{3, 4, 5}, ids)
	assert.Equal(t, board[colID].Count, total)
}

func TestSavedFilterMirror_T4_NoMirrorWithTwoManualKanbans(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()
	u := &user.User{ID: 1}

	view, todoID := createMirrorFilter(t, s, u, "project = 1 && done = false")
	addMirrorColumn(t, s, u, view, " TESTBUCKET2 ")
	_, err := s.Insert(&ProjectView{
		ProjectID:               1,
		Title:                   "second kanban",
		ViewKind:                ProjectViewKindKanban,
		BucketConfigurationMode: BucketConfigurationModeManual,
	})
	require.NoError(t, err)

	readMirrorBoard(t, s, u, view)
	for _, taskID := range []int64{3, 4, 5} {
		assert.Equal(t, todoID, mirrorRow(t, s, view.ID, taskID))
	}
}

func TestSavedFilterMirror_T5_IsMirroredFilterView(t *testing.T) {
	filterProjectID := getProjectIDFromSavedFilterID(1)
	assert.True(t, isMirroredFilterView(&ProjectView{
		ProjectID:               filterProjectID,
		ViewKind:                ProjectViewKindKanban,
		BucketConfigurationMode: BucketConfigurationModeManual,
	}))
	assert.False(t, isMirroredFilterView(&ProjectView{
		ProjectID:               filterProjectID,
		ViewKind:                ProjectViewKindKanban,
		BucketConfigurationMode: BucketConfigurationModeFilter,
	}))
	assert.False(t, isMirroredFilterView(&ProjectView{
		ID:                      4,
		ProjectID:               1,
		ViewKind:                ProjectViewKindKanban,
		BucketConfigurationMode: BucketConfigurationModeManual,
	}))
}

func TestSavedFilterMirror_T6_DropMovesProjectCard(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()
	u := &user.User{ID: 1}

	view, _ := createMirrorFilter(t, s, u, "project = 1 && done = false")
	col1 := addMirrorColumn(t, s, u, view, "testbucket1")
	addMirrorColumn(t, s, u, view, "testbucket3")

	tb, err := moveMirrorTask(s, u, view, 3, col1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), mirrorRow(t, s, 4, 3))
	assert.Equal(t, col1, tb.BucketID)
	assert.Equal(t, col1, tb.Bucket.ID)
	assert.Equal(t, col1, mirrorRow(t, s, view.ID, 3))
}

func TestSavedFilterMirror_T7_DropIntoStatusDoneCompletes(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()
	u := &user.User{ID: 1}

	view, _ := createMirrorFilter(t, s, u, "project = 1 && done = false")
	col1 := addMirrorColumn(t, s, u, view, "testbucket1")
	col3 := addMirrorColumn(t, s, u, view, "testbucket3")

	tb, err := moveMirrorTask(s, u, view, 3, col3)
	require.NoError(t, err)
	assert.True(t, tb.Task.Done)
	assert.Equal(t, int64(3), mirrorRow(t, s, 4, 3))

	tb, err = moveMirrorTask(s, u, view, 3, col1)
	require.NoError(t, err)
	assert.False(t, tb.Task.Done)
	assert.Equal(t, int64(1), mirrorRow(t, s, 4, 3))
}

func TestSavedFilterMirror_T8_RepeatingTaskRerouted(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()
	u := &user.User{ID: 1}

	view, _ := createMirrorFilter(t, s, u, "project = 1 && done = false")
	col1 := addMirrorColumn(t, s, u, view, "testbucket1")
	col3 := addMirrorColumn(t, s, u, view, "testbucket3")

	before := &Task{ID: 28}
	require.NoError(t, before.ReadOne(s, u))

	tb, err := moveMirrorTask(s, u, view, 28, col3)
	require.NoError(t, err)
	assert.False(t, tb.Task.Done)
	assert.True(t, tb.Task.DueDate.After(before.DueDate))
	assert.Equal(t, int64(1), mirrorRow(t, s, 4, 28))
	assert.Equal(t, col1, tb.BucketID)
	assert.Equal(t, col1, mirrorRow(t, s, view.ID, 28))
}

func TestSavedFilterMirror_T9_UnmatchedDropIsFilterOnly(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()
	u := &user.User{ID: 1}

	view, _ := createMirrorFilter(t, s, u, "project = 1 && done = false")
	testingID := addMirrorColumn(t, s, u, view, "Testing")

	_, err := moveMirrorTask(s, u, view, 3, testingID)
	require.NoError(t, err)
	assert.Equal(t, testingID, mirrorRow(t, s, view.ID, 3))
	assert.Equal(t, int64(2), mirrorRow(t, s, 4, 3))

	readMirrorBoard(t, s, u, view)
	assert.Equal(t, testingID, mirrorRow(t, s, view.ID, 3))
}

func TestSavedFilterMirror_T10_SameColumnDropForwardsWithCount(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()
	u := &user.User{ID: 1}

	view, _ := createMirrorFilter(t, s, u, "project = 1 && done = false")
	col1 := addMirrorColumn(t, s, u, view, "testbucket1")
	setMirrorRow(t, s, view.ID, 3, col1)
	setMirrorRow(t, s, view.ID, 4, col1)

	tb, err := moveMirrorTask(s, u, view, 3, col1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), mirrorRow(t, s, 4, 3))
	assert.Equal(t, col1, tb.BucketID)
	require.NotNil(t, tb.Bucket)
	// Exact, so a plain "+1" on an unset count (1) can't pass for the scoped count (tasks 3 and 4).
	assert.Equal(t, int64(2), tb.Bucket.Count)
}

func TestSavedFilterMirror_T11_ProjectBucketLimitApplies(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()
	u := &user.User{ID: 1}

	view, _ := createMirrorFilter(t, s, u, "project = 1 && done = false")
	col2 := addMirrorColumn(t, s, u, view, "testbucket2")

	_, err := moveMirrorTask(s, u, view, 1, col2)
	require.Error(t, err)
	assert.True(t, IsErrBucketLimitExceeded(err))
	assert.Equal(t, int64(1), mirrorRow(t, s, 4, 1))
}

func TestSavedFilterMirror_T12_ListenerKeepsMatchedCard(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	u := &user.User{ID: 1}

	view, _ := createMirrorFilter(t, s, u, "project = 1")
	col3 := addMirrorColumn(t, s, u, view, "testbucket3")
	view.DoneBucketID = col3
	_, err := s.ID(view.ID).Cols("done_bucket_id").Update(view)
	require.NoError(t, err)

	readMirrorBoard(t, s, u, view)
	assert.Equal(t, col3, mirrorRow(t, s, view.ID, 6))
	require.NoError(t, s.Commit())
	_ = s.Close()

	events.TestListener(t, &TaskUpdatedEvent{
		Task: &Task{
			ID:        6,
			ProjectID: 1,
		},
		Doer: u,
	}, &UpdateTaskInSavedFilterViews{})

	db.AssertExists(t, "task_buckets", map[string]interface{}{
		"task_id":         6,
		"project_view_id": view.ID,
		"bucket_id":       col3,
	}, false)
}

func TestSavedFilterMirror_T13_StaleMoveIsSkipped(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()
	u := &user.User{ID: 1}

	view, todoID := createMirrorFilter(t, s, u, "project = 1 && done = false")
	colID := addMirrorColumn(t, s, u, view, " TESTBUCKET2 ")
	customID := addMirrorColumn(t, s, u, view, "Custom")
	setMirrorRow(t, s, view.ID, 3, customID)

	require.NoError(t, applyMirrorMoves(s, view.ID, []mirrorMove{
		{
			taskID: 3,
			from:   todoID,
			to:     colID,
		},
	}))
	assert.Equal(t, customID, mirrorRow(t, s, view.ID, 3))
}

func TestSavedFilterMirror_T14_NoOpDropDispatchesNoEvent(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()
	u := &user.User{ID: 1}

	view, _ := createMirrorFilter(t, s, u, "project = 1 && done = false")
	colID := addMirrorColumn(t, s, u, view, " TESTBUCKET2 ")
	readMirrorBoard(t, s, u, view)
	require.Equal(t, colID, mirrorRow(t, s, view.ID, 3))

	events.ClearDispatchedEvents()
	tb, err := moveMirrorTask(s, u, view, 3, colID)
	require.NoError(t, err)
	require.NoError(t, s.Commit())
	events.DispatchPending(context.Background(), s)

	assert.Nil(t, tb.Task)
	assert.Zero(t, events.CountDispatchedEvents((&TaskUpdatedEvent{}).Name()))
}

func TestSavedFilterMirror_B5_NewProjectIdeaColumn(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()
	u := &user.User{ID: 1}

	project, _, _ := newIdeaBoard(t, s, u)
	// Filter rows are backfilled when the filter is created; later tasks arrive via the async listener.
	id := newBoardTask(t, s, u, project.ID, &Task{})
	view, todoID := createMirrorFilter(t, s, u, "project = "+strconv.FormatInt(project.ID, 10))

	// A new filter board has an Idea column, but keeps To-Do as its default.
	buckets := []*Bucket{}
	require.NoError(t, s.Where("project_view_id = ?", view.ID).OrderBy("position asc, id asc").Find(&buckets))
	require.Len(t, buckets, 4)
	assert.Equal(t, "Idea", buckets[0].Title)
	assert.Equal(t, "To-Do", buckets[1].Title)
	assert.Equal(t, buckets[1].ID, todoID)

	board := readMirrorBoard(t, s, u, view)
	assert.Equal(t, []int64{id}, mirrorTaskIDs(board[buckets[0].ID]))
	assert.Empty(t, mirrorTaskIDs(board[todoID]))
}
