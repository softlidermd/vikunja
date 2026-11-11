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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"xorm.io/builder"
	"xorm.io/xorm"
)

type taskBucketFor20261111120000 struct {
	BucketID      int64 `xorm:"bigint not null index"`
	TaskID        int64 `xorm:"bigint not null index unique(task_view)"`
	ProjectViewID int64 `xorm:"bigint not null index unique(task_view)"`
}

func (taskBucketFor20261111120000) TableName() string {
	return "task_buckets"
}

type projectsFor20261111120000 struct {
	ID      int64 `xorm:"bigint autoincr not null unique pk"`
	OwnerID int64 `xorm:"bigint not null"`
}

func (projectsFor20261111120000) TableName() string {
	return "projects"
}

type savedFiltersFor20261111120000 struct {
	ID      int64 `xorm:"bigint autoincr not null unique pk"`
	OwnerID int64 `xorm:"bigint not null"`
}

func (savedFiltersFor20261111120000) TableName() string {
	return "saved_filters"
}

func bucketsOf20261111120000(t *testing.T, x *xorm.Engine, viewID int64) []*bucket20261111120000 {
	t.Helper()
	buckets := []*bucket20261111120000{}
	require.NoError(t, x.Where(builder.Eq{"project_view_id": viewID}).OrderBy("position, id").Find(&buckets))
	return buckets
}

func defaultOf20261111120000(t *testing.T, x *xorm.Engine, viewID int64) int64 {
	t.Helper()
	view := &projectView20261111120000{}
	has, err := x.Where(builder.Eq{"id": viewID}).Get(view)
	require.NoError(t, err)
	require.True(t, has)
	return view.DefaultBucketID
}

func TestIdeaBucketBackfill20261111120000(t *testing.T) {
	x, err := db.CreateTestEngine()
	require.NoError(t, err)

	tables := []any{
		projectView20261111120000{},
		bucket20261111120000{},
		taskBucketFor20261111120000{},
		projectsFor20261111120000{},
		savedFiltersFor20261111120000{},
	}
	// x is the process-global test engine; these minimal stub tables would leak into other tests.
	t.Cleanup(func() {
		require.NoError(t, x.DropTables(tables...))
	})
	require.NoError(t, x.DropTables(tables...))
	require.NoError(t, x.Sync2(tables...))

	// view_kind: 0 = list, 3 = kanban. bucket_configuration_mode: 1 = manual, 2 = filter.
	_, err = x.Insert(
		&projectsFor20261111120000{ID: 1, OwnerID: 42},
		&projectsFor20261111120000{ID: 2, OwnerID: 42},
		&projectsFor20261111120000{ID: 3, OwnerID: 42},
		&projectsFor20261111120000{ID: 4, OwnerID: 42},
		&projectsFor20261111120000{ID: 5, OwnerID: 42},
		&projectsFor20261111120000{ID: 6, OwnerID: 42},
		&savedFiltersFor20261111120000{ID: 1, OwnerID: 7},
		&savedFiltersFor20261111120000{ID: 2, OwnerID: 7},

		// (a) status board with a task in To-Do, plus a list view and a filter-mode kanban that don't count
		&projectView20261111120000{ID: 1, ProjectID: 1, ViewKind: 3, BucketConfigurationMode: 1, DefaultBucketID: 11, DoneBucketID: 13},
		&projectView20261111120000{ID: 2, ProjectID: 1, ViewKind: 0},
		&projectView20261111120000{ID: 3, ProjectID: 1, ViewKind: 3, BucketConfigurationMode: 2},
		&bucket20261111120000{ID: 11, Title: "To-Do", ProjectViewID: 1, Position: 100, CreatedByID: 42},
		&bucket20261111120000{ID: 12, Title: "Doing", ProjectViewID: 1, Position: 200, CreatedByID: 42},
		&bucket20261111120000{ID: 13, Title: "Done", ProjectViewID: 1, Position: 300, CreatedByID: 42},
		&taskBucketFor20261111120000{TaskID: 1, ProjectViewID: 1, BucketID: 11},

		// (b) an existing " IDEA" bucket that is not the default
		&projectView20261111120000{ID: 4, ProjectID: 2, ViewKind: 3, BucketConfigurationMode: 1, DefaultBucketID: 21},
		&bucket20261111120000{ID: 20, Title: " IDEA", ProjectViewID: 4, Position: 400, CreatedByID: 42},
		&bucket20261111120000{ID: 21, Title: "To-Do", ProjectViewID: 4, Position: 100, CreatedByID: 42},

		// (c) two manual kanban views: ambiguous
		&projectView20261111120000{ID: 5, ProjectID: 3, ViewKind: 3, BucketConfigurationMode: 1, DefaultBucketID: 30},
		&projectView20261111120000{ID: 6, ProjectID: 3, ViewKind: 3, BucketConfigurationMode: 1, DefaultBucketID: 31},
		&bucket20261111120000{ID: 30, Title: "To-Do", ProjectViewID: 5, Position: 100, CreatedByID: 42},
		&bucket20261111120000{ID: 31, Title: "To-Do", ProjectViewID: 6, Position: 100, CreatedByID: 42},

		// (d) filter board of filter 1 (project_id -2)
		&projectView20261111120000{ID: 7, ProjectID: -2, ViewKind: 3, BucketConfigurationMode: 1, DefaultBucketID: 41},
		&bucket20261111120000{ID: 40, Title: "To-Do", ProjectViewID: 7, Position: 100, CreatedByID: 7},
		&bucket20261111120000{ID: 41, Title: "Doing", ProjectViewID: 7, Position: 200, CreatedByID: 7},

		// (e) no kanban at all
		&projectView20261111120000{ID: 8, ProjectID: 4, ViewKind: 0},

		// (f) filter board of filter 2 (project_id -3) without an explicit default; positions down to 0
		&projectView20261111120000{ID: 9, ProjectID: -3, ViewKind: 3, BucketConfigurationMode: 1},
		&bucket20261111120000{ID: 50, Title: "To-Do", ProjectViewID: 9, Position: 0, CreatedByID: 7},
		&bucket20261111120000{ID: 51, Title: "Done", ProjectViewID: 9, Position: 10, CreatedByID: 7},

		// (g) status board without an explicit default
		&projectView20261111120000{ID: 10, ProjectID: 5, ViewKind: 3, BucketConfigurationMode: 1},
		&bucket20261111120000{ID: 60, Title: "To-Do", ProjectViewID: 10, Position: 100, CreatedByID: 42},
		&bucket20261111120000{ID: 61, Title: "Done", ProjectViewID: 10, Position: 200, CreatedByID: 42},

		// Empty status board, and the Favorites pseudo project (-1): both untouched
		&projectView20261111120000{ID: 11, ProjectID: 6, ViewKind: 3, BucketConfigurationMode: 1},
		&projectView20261111120000{ID: 12, ProjectID: -1, ViewKind: 3, BucketConfigurationMode: 1, DefaultBucketID: 70},
		&bucket20261111120000{ID: 70, Title: "To-Do", ProjectViewID: 12, Position: 100, CreatedByID: 42},
	)
	require.NoError(t, err)

	require.NoError(t, addIdeaBuckets20261111120000(x))

	t.Run("a: status board gets Idea leftmost as default, task stays", func(t *testing.T) {
		buckets := bucketsOf20261111120000(t, x, 1)
		require.Len(t, buckets, 4)
		idea := buckets[0]
		assert.Equal(t, "Idea", idea.Title)
		assert.InDelta(t, 50, idea.Position, 0)
		assert.Equal(t, int64(42), idea.CreatedByID)
		assert.False(t, idea.Created.IsZero())
		assert.False(t, idea.Updated.IsZero())
		assert.Equal(t, idea.ID, defaultOf20261111120000(t, x, 1))

		taskBucket := &taskBucketFor20261111120000{}
		has, err := x.Where(builder.Eq{"task_id": 1, "project_view_id": 1}).Get(taskBucket)
		require.NoError(t, err)
		require.True(t, has)
		assert.Equal(t, int64(11), taskBucket.BucketID)

		assert.Empty(t, bucketsOf20261111120000(t, x, 2))
		assert.Empty(t, bucketsOf20261111120000(t, x, 3))
	})

	t.Run("b: an existing IDEA bucket becomes the default", func(t *testing.T) {
		assert.Len(t, bucketsOf20261111120000(t, x, 4), 2)
		assert.Equal(t, int64(20), defaultOf20261111120000(t, x, 4))
	})

	t.Run("c: two manual kanban views stay unchanged", func(t *testing.T) {
		assert.Len(t, bucketsOf20261111120000(t, x, 5), 1)
		assert.Len(t, bucketsOf20261111120000(t, x, 6), 1)
		assert.Equal(t, int64(30), defaultOf20261111120000(t, x, 5))
		assert.Equal(t, int64(31), defaultOf20261111120000(t, x, 6))
	})

	t.Run("d: filter board gets Idea leftmost, default unchanged", func(t *testing.T) {
		buckets := bucketsOf20261111120000(t, x, 7)
		require.Len(t, buckets, 3)
		assert.Equal(t, "Idea", buckets[0].Title)
		assert.InDelta(t, 50, buckets[0].Position, 0)
		assert.Equal(t, int64(7), buckets[0].CreatedByID)
		assert.Equal(t, int64(41), defaultOf20261111120000(t, x, 7))
	})

	t.Run("e: project without kanban stays unchanged", func(t *testing.T) {
		assert.Empty(t, bucketsOf20261111120000(t, x, 8))
		assert.Zero(t, defaultOf20261111120000(t, x, 8))
	})

	t.Run("f: filter board without default pins To-Do, then Idea goes left of it", func(t *testing.T) {
		buckets := bucketsOf20261111120000(t, x, 9)
		require.Len(t, buckets, 3)
		assert.Equal(t, "Idea", buckets[0].Title)
		assert.InDelta(t, -1, buckets[0].Position, 0)
		assert.Equal(t, int64(50), buckets[1].ID)
		assert.Equal(t, int64(50), defaultOf20261111120000(t, x, 9))
	})

	t.Run("g: status board without default gets Idea as default", func(t *testing.T) {
		buckets := bucketsOf20261111120000(t, x, 10)
		require.Len(t, buckets, 3)
		assert.Equal(t, "Idea", buckets[0].Title)
		assert.Equal(t, buckets[0].ID, defaultOf20261111120000(t, x, 10))
	})

	t.Run("empty board and Favorites stay unchanged", func(t *testing.T) {
		assert.Empty(t, bucketsOf20261111120000(t, x, 11))
		assert.Zero(t, defaultOf20261111120000(t, x, 11))
		assert.Len(t, bucketsOf20261111120000(t, x, 12), 1)
		assert.Equal(t, int64(70), defaultOf20261111120000(t, x, 12))
	})

	t.Run("a second run changes nothing", func(t *testing.T) {
		before, err := x.Count(&bucket20261111120000{})
		require.NoError(t, err)
		views := []*projectView20261111120000{}
		require.NoError(t, x.OrderBy("id").Find(&views))

		require.NoError(t, addIdeaBuckets20261111120000(x))

		after, err := x.Count(&bucket20261111120000{})
		require.NoError(t, err)
		assert.Equal(t, before, after)
		viewsAfter := []*projectView20261111120000{}
		require.NoError(t, x.OrderBy("id").Find(&viewsAfter))
		assert.Equal(t, views, viewsAfter)
	})
}
