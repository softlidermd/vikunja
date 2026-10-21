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

func TestAddLabelsViews20261021120000(t *testing.T) {
	x, err := db.CreateTestEngine()
	require.NoError(t, err)
	// The full model structs: a partial Sync2 on the shared engine would drop the tables' indexes.
	require.NoError(t, x.Sync2(&models.Project{}, &models.SavedFilter{}, &models.ProjectView{}))

	const title = "labels view test 20261021120000"
	viewOwnerIDs := []int64{}
	t.Cleanup(func() {
		_, err := x.Where(builder.Eq{"title": title}).Delete(&models.Project{})
		require.NoError(t, err)
		_, err = x.Where(builder.Eq{"title": title}).Delete(&models.SavedFilter{})
		require.NoError(t, err)
		_, err = x.Where(builder.Or(
			builder.In("project_id", viewOwnerIDs),
			builder.Eq{"view_kind": models.ProjectViewKindLabels},
		)).Delete(&models.ProjectView{})
		require.NoError(t, err)
	})

	newProject := func() int64 {
		p := &models.Project{
			Title:   title,
			OwnerID: 1,
		}
		_, err := x.Insert(p)
		require.NoError(t, err)
		viewOwnerIDs = append(viewOwnerIDs, p.ID)
		return p.ID
	}
	newFilter := func() int64 {
		f := &models.SavedFilter{
			Title:   title,
			OwnerID: 1,
			Filters: &models.TaskCollection{},
		}
		_, err := x.Insert(f)
		require.NoError(t, err)
		viewOwnerIDs = append(viewOwnerIDs, -f.ID-1)
		return -f.ID - 1
	}
	addViews := func(ownerID int64, kind models.ProjectViewKind, positions ...float64) {
		for _, pos := range positions {
			_, err := x.Insert(&models.ProjectView{
				Title:     "View",
				ProjectID: ownerID,
				ViewKind:  kind,
				Position:  pos,
			})
			require.NoError(t, err)
		}
	}

	manyViews := newProject()
	addViews(manyViews, models.ProjectViewKindList, 100, 200, 300, 400)
	gappedViews := newProject()
	addViews(gappedViews, models.ProjectViewKindList, 100, 900)
	hasLabels := newProject()
	addViews(hasLabels, models.ProjectViewKindList, 100)
	addViews(hasLabels, models.ProjectViewKindLabels, 250)
	noViews := newProject()
	filterWithViews := newFilter()
	addViews(filterWithViews, models.ProjectViewKindList, 100, 200, 300, 400)
	filterNoViews := newFilter()
	orphan := int64(1) << 40
	viewOwnerIDs = append(viewOwnerIDs, orphan)
	addViews(orphan, models.ProjectViewKindList, 100)

	require.NoError(t, addLabelsViews20261021120000(x))
	require.NoError(t, addLabelsViews20261021120000(x))

	want := map[int64][]float64{
		manyViews:       {500},
		gappedViews:     {1000},
		hasLabels:       {250},
		noViews:         {100},
		filterWithViews: {500},
		filterNoViews:   {100},
		orphan:          nil,
	}
	for ownerID, positions := range want {
		views := []*models.ProjectView{}
		require.NoError(t, x.Where(builder.Eq{
			"project_id": ownerID,
			"view_kind":  models.ProjectViewKindLabels,
		}).Find(&views))
		got := []float64(nil)
		for _, v := range views {
			got = append(got, v.Position)
		}
		assert.Equalf(t, positions, got, "labels views of project %d", ownerID)
	}
}
