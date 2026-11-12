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
	"time"

	"code.vikunja.io/api/pkg/db"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"xorm.io/builder"
)

type usersFor20261112120000 struct {
	ID               int64  `xorm:"bigint autoincr not null unique pk"`
	Username         string `xorm:"varchar(250) not null"`
	DefaultProjectID *int64 `xorm:"bigint null"`
	BotOwnerID       *int64 `xorm:"bigint null"`
}

func (usersFor20261112120000) TableName() string {
	return "users"
}

type projectsFor20261112120000 struct {
	ID         int64  `xorm:"bigint autoincr not null unique pk"`
	Title      string `xorm:"varchar(250) not null"`
	OwnerID    int64  `xorm:"bigint INDEX not null"`
	IsArchived bool   `xorm:"not null default false"`
}

func (projectsFor20261112120000) TableName() string {
	return "projects"
}

type tasksFor20261112120000 struct {
	ID        int64     `xorm:"bigint autoincr not null unique pk"`
	Title     string    `xorm:"TEXT not null"`
	ProjectID int64     `xorm:"bigint INDEX not null"`
	DeletedAt time.Time `xorm:"datetime null INDEX"`
}

func (tasksFor20261112120000) TableName() string {
	return "tasks"
}

type labelsBefore20261112120000 struct {
	ID          int64     `xorm:"bigint autoincr not null unique pk"`
	Title       string    `xorm:"varchar(250) not null"`
	Description string    `xorm:"longtext null"`
	HexColor    string    `xorm:"varchar(6) null"`
	CreatedByID int64     `xorm:"bigint not null"`
	Created     time.Time `xorm:"not null"`
	Updated     time.Time `xorm:"not null"`
}

func (labelsBefore20261112120000) TableName() string {
	return "labels"
}

type labelTasksBefore20261112120000 struct {
	ID      int64     `xorm:"bigint autoincr not null unique pk"`
	TaskID  int64     `xorm:"bigint INDEX not null"`
	LabelID int64     `xorm:"bigint INDEX not null"`
	Created time.Time `xorm:"not null"`
}

func (labelTasksBefore20261112120000) TableName() string {
	return "label_tasks"
}

func TestMigrateLabelsToProjects(t *testing.T) {
	x, err := db.CreateTestEngine()
	require.NoError(t, err)

	tables := []any{
		usersFor20261112120000{},
		projectsFor20261112120000{},
		tasksFor20261112120000{},
		labelsBefore20261112120000{},
		labelTasksBefore20261112120000{},
	}
	// x is the process-global test engine; these tables would leak into other tests.
	t.Cleanup(func() {
		require.NoError(t, x.DropTables(tables...))
	})
	require.NoError(t, x.DropTables(tables...))
	require.NoError(t, x.Sync2(tables...))

	before := tablesByName20261111120100(t, x)
	require.NotEmpty(t, before["labels"].Indexes)
	require.NotEmpty(t, before["label_tasks"].Indexes)

	// The columns of an earlier, interrupted run, so G can already have its project.
	require.NoError(t, partialSync(x, labels20261112120000{}))

	ptr := func(v int64) *int64 { return &v }
	_, err = x.Insert(
		&usersFor20261112120000{ID: 1, Username: "owner a"},
		&usersFor20261112120000{ID: 2, Username: "b", DefaultProjectID: ptr(21)},
		&usersFor20261112120000{ID: 3, Username: "c", DefaultProjectID: ptr(1)},
		&usersFor20261112120000{ID: 4, Username: "d"},
		&usersFor20261112120000{ID: 5, Username: "bot of d", BotOwnerID: ptr(4)},
		&usersFor20261112120000{ID: 6, Username: "e"},
		&usersFor20261112120000{ID: 7, Username: "f"},
	)
	require.NoError(t, err)
	_, err = x.Insert(
		&projectsFor20261112120000{ID: 1, Title: "p1", OwnerID: 1},
		&projectsFor20261112120000{ID: 2, Title: "p2", OwnerID: 1},
		&projectsFor20261112120000{ID: 20, Title: "b other", OwnerID: 2},
		&projectsFor20261112120000{ID: 21, Title: "b default", OwnerID: 2},
		&projectsFor20261112120000{ID: 30, Title: "c archived", OwnerID: 3, IsArchived: true},
		&projectsFor20261112120000{ID: 31, Title: "c", OwnerID: 3},
		&projectsFor20261112120000{ID: 40, Title: "d", OwnerID: 4},
		&projectsFor20261112120000{ID: 50, Title: "f", OwnerID: 7},
	)
	require.NoError(t, err)
	deletedAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	_, err = x.Insert(
		&tasksFor20261112120000{ID: 1, Title: "a in p1", ProjectID: 1},
		&tasksFor20261112120000{ID: 2, Title: "a in p1", ProjectID: 1},
		&tasksFor20261112120000{ID: 3, Title: "a in p2", ProjectID: 2},
		&tasksFor20261112120000{ID: 4, Title: "a in p2, soft-deleted", ProjectID: 2, DeletedAt: deletedAt},
		&tasksFor20261112120000{ID: 5, Title: "h in p1", ProjectID: 1},
		&tasksFor20261112120000{ID: 6, Title: "h in p2", ProjectID: 2},
		&tasksFor20261112120000{ID: 7, Title: "h in p2", ProjectID: 2},
		&tasksFor20261112120000{ID: 8, Title: "g in p1", ProjectID: 1},
		&tasksFor20261112120000{ID: 9, Title: "in a missing project", ProjectID: 999},
	)
	require.NoError(t, err)

	created := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	updated := time.Date(2021, 1, 2, 3, 4, 5, 0, time.UTC)
	// Ids are left to the database: the migration's clone inserts share its sequence.
	newLabel := func(title string, createdBy, projectID int64) int64 {
		l := &labelRow20261112120000{
			Title:       title,
			Description: "description of " + title,
			HexColor:    "e8e8e8",
			CreatedByID: createdBy,
			ProjectID:   projectID,
			Created:     created,
			Updated:     updated,
		}
		_, err := x.Insert(l)
		require.NoError(t, err)
		return l.ID
	}
	labelA := newLabel("a: used in p1 and p2", 2, 0)
	labelH := newLabel("h: used most in p2", 1, 0)
	labelB := newLabel("b: unused, creator owns default", 2, 0)
	labelC := newLabel("c: unused, default is foreign", 3, 0)
	labelD := newLabel("d: unused, created by a bot", 5, 0)
	labelE := newLabel("e: unused, creator owns nothing", 6, 0)
	labelF := newLabel("f: only on an orphan row", 7, 0)
	labelG := newLabel("g: already migrated", 1, 1)

	newLabelTask := func(taskID, labelID int64) int64 {
		lt := &labelTasksBefore20261112120000{
			TaskID:  taskID,
			LabelID: labelID,
			Created: created,
		}
		_, err := x.Insert(lt)
		require.NoError(t, err)
		return lt.ID
	}
	a1 := newLabelTask(1, labelA)
	a2 := newLabelTask(2, labelA)
	a3 := newLabelTask(3, labelA)
	a4 := newLabelTask(4, labelA)
	h5 := newLabelTask(5, labelH)
	h6 := newLabelTask(6, labelH)
	h7 := newLabelTask(7, labelH)
	g8 := newLabelTask(8, labelG)
	orphanMissingTask := newLabelTask(12345, labelF)
	orphanMissingProject := newLabelTask(9, labelA)
	orphanMissingLabel := newLabelTask(1, 999999)

	require.NoError(t, migrateLabelsToProjects20261112120000(x))

	getLabel := func(id int64) *labelRow20261112120000 {
		l := &labelRow20261112120000{}
		has, err := x.ID(id).Get(l)
		require.NoError(t, err)
		require.Truef(t, has, "label %d is gone", id)
		return l
	}
	cloneOf := func(rootID int64) *labelRow20261112120000 {
		clones := []*labelRow20261112120000{}
		require.NoError(t, x.Where(builder.Eq{"cloned_from_id": rootID}).Find(&clones))
		require.Lenf(t, clones, 1, "clones of label %d", rootID)
		return clones[0]
	}
	labelOfRow := func(id int64) int64 {
		lt := &labelTask20261112120000{}
		has, err := x.ID(id).Get(lt)
		require.NoError(t, err)
		require.Truef(t, has, "label_tasks row %d is gone", id)
		return lt.LabelID
	}

	t.Run("A keeps its id in the tied lowest project, the clone owns both p2 rows", func(t *testing.T) {
		a := getLabel(labelA)
		assert.Equal(t, int64(1), a.ProjectID)
		assert.Equal(t, int64(0), a.ClonedFromID)

		clone := cloneOf(labelA)
		assert.Equal(t, int64(2), clone.ProjectID)
		assert.Equal(t, a.Title, clone.Title)
		assert.Equal(t, a.Description, clone.Description)
		assert.Equal(t, a.HexColor, clone.HexColor)
		assert.Equal(t, int64(2), clone.CreatedByID)
		assert.Equal(t, created.Unix(), clone.Created.Unix())
		assert.Equal(t, updated.Unix(), clone.Updated.Unix())

		assert.Equal(t, labelA, labelOfRow(a1))
		assert.Equal(t, labelA, labelOfRow(a2))
		assert.Equal(t, clone.ID, labelOfRow(a3))
		assert.Equal(t, clone.ID, labelOfRow(a4))
	})
	t.Run("H keeps its id in its most-used project", func(t *testing.T) {
		assert.Equal(t, int64(2), getLabel(labelH).ProjectID)
		clone := cloneOf(labelH)
		assert.Equal(t, int64(1), clone.ProjectID)
		assert.Equal(t, clone.ID, labelOfRow(h5))
		assert.Equal(t, labelH, labelOfRow(h6))
		assert.Equal(t, labelH, labelOfRow(h7))
	})
	t.Run("unused labels go to a project the human creator owns", func(t *testing.T) {
		assert.Equal(t, int64(21), getLabel(labelB).ProjectID, "B: owned default project")
		assert.Equal(t, int64(31), getLabel(labelC).ProjectID, "C: first non-archived owned project")
		assert.Equal(t, int64(40), getLabel(labelD).ProjectID, "D: the bot owner's project")
		assert.Equal(t, int64(50), getLabel(labelF).ProjectID, "F: its only row was an orphan")
		exists, err := x.ID(labelE).Exist(&labelRow20261112120000{})
		require.NoError(t, err)
		assert.False(t, exists, "E: its creator owns nothing")
	})
	t.Run("G is untouched", func(t *testing.T) {
		g := getLabel(labelG)
		assert.Equal(t, int64(1), g.ProjectID)
		assert.Equal(t, "g: already migrated", g.Title)
		assert.Equal(t, labelG, labelOfRow(g8))
		count, err := x.Where(builder.Eq{"cloned_from_id": labelG}).Count(&labelRow20261112120000{})
		require.NoError(t, err)
		assert.Zero(t, count)
	})
	t.Run("orphan rows are gone", func(t *testing.T) {
		for _, id := range []int64{orphanMissingTask, orphanMissingProject, orphanMissingLabel} {
			exists, err := x.ID(id).Exist(&labelTask20261112120000{})
			require.NoError(t, err)
			assert.Falsef(t, exists, "label_tasks row %d", id)
		}
		count, err := x.Count(&labelTask20261112120000{})
		require.NoError(t, err)
		assert.Equal(t, int64(8), count)
	})
	t.Run("indexes survive", func(t *testing.T) {
		after := tablesByName20261111120100(t, x)
		for _, name := range []string{"labels", "label_tasks"} {
			for indexName, index := range before[name].Indexes {
				preserved, found := after[name].Indexes[indexName]
				require.Truef(t, found, "migration dropped index %s on %s", indexName, name)
				assert.Equal(t, index.Type, preserved.Type)
				assert.Equal(t, index.Cols, preserved.Cols)
			}
		}
	})
	t.Run("a second run changes nothing", func(t *testing.T) {
		snapshot := func() ([]*labelRow20261112120000, []*labelTask20261112120000) {
			labels := []*labelRow20261112120000{}
			require.NoError(t, x.OrderBy("id").Find(&labels))
			labelTasks := []*labelTask20261112120000{}
			require.NoError(t, x.OrderBy("id").Find(&labelTasks))
			return labels, labelTasks
		}
		labelsBefore, labelTasksBefore := snapshot()
		require.NoError(t, migrateLabelsToProjects20261112120000(x))
		labelsAfter, labelTasksAfter := snapshot()
		assert.Equal(t, labelsBefore, labelsAfter)
		assert.Equal(t, labelTasksBefore, labelTasksAfter)
	})
}
