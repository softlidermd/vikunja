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
)

type projects20261021120000 struct {
	ID int64 `xorm:"autoincr not null unique pk"`
}

func (projects20261021120000) TableName() string {
	return "projects"
}

type savedFilters20261021120000 struct {
	ID int64 `xorm:"autoincr not null unique pk"`
}

func (savedFilters20261021120000) TableName() string {
	return "saved_filters"
}

func TestAddLabelsViews20261021120000(t *testing.T) {
	x, err := db.CreateTestEngine()
	require.NoError(t, err)
	require.NoError(t, x.Sync2(projectView20261021120000{}, projects20261021120000{}, savedFilters20261021120000{}))

	projectIDs := []int64{9001, 9002, 9003, -9001, 9999} // -9001 is saved filter 9000, 9999 has no project
	t.Cleanup(func() {
		_, err := x.In("id", 9001, 9002, 9003).Delete(&projects20261021120000{})
		require.NoError(t, err)
		_, err = x.ID(9000).Delete(&savedFilters20261021120000{})
		require.NoError(t, err)
		_, err = x.Where(builder.Or(builder.In("project_id", projectIDs), builder.Eq{"view_kind": viewKindLabels20261021120000})).Delete(&projectView20261021120000{})
		require.NoError(t, err)
	})

	for _, id := range []int64{9001, 9002, 9003} {
		_, err = x.Insert(&projects20261021120000{ID: id})
		require.NoError(t, err)
	}
	_, err = x.Insert(&savedFilters20261021120000{ID: 9000})
	require.NoError(t, err)

	seed := map[int64][]float64{
		9001:  {100, 200, 300, 400},
		9002:  {100, 900},
		9003:  {100},
		-9001: {100, 200, 300, 400},
		9999:  {100},
	}
	for projectID, positions := range seed {
		for _, pos := range positions {
			_, err = x.Insert(&projectView20261021120000{Title: "View", ProjectID: projectID, Position: pos})
			require.NoError(t, err)
		}
	}
	_, err = x.Insert(&projectView20261021120000{Title: "Mine", ProjectID: 9003, ViewKind: viewKindLabels20261021120000, Position: 250})
	require.NoError(t, err)

	require.NoError(t, addLabelsViews20261021120000(x))
	require.NoError(t, addLabelsViews20261021120000(x))

	want := map[int64][]float64{9001: {500}, 9002: {1000}, 9003: {250}, -9001: {500}, 9999: nil}
	for projectID, positions := range want {
		views := []*projectView20261021120000{}
		require.NoError(t, x.Where("project_id = ? AND view_kind = ?", projectID, viewKindLabels20261021120000).Find(&views))
		got := []float64(nil)
		for _, v := range views {
			got = append(got, v.Position)
		}
		assert.Equalf(t, positions, got, "labels views of project %d", projectID)
	}
}
