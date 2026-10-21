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
	"fmt"
	"time"

	"src.techknowlogick.com/xormigrate"
	"xorm.io/xorm"
)

const viewKindLabels20261021120000 = 100

// Only the columns an insert needs: the rest are nullable or carry a default.
type projectView20261021120000 struct {
	ID        int64     `xorm:"autoincr not null unique pk"`
	Title     string    `xorm:"varchar(255) not null"`
	ProjectID int64     `xorm:"not null index"`
	ViewKind  int       `xorm:"not null"`
	Position  float64   `xorm:"double null"`
	Updated   time.Time `xorm:"updated not null"`
	Created   time.Time `xorm:"created not null"`
}

func (projectView20261021120000) TableName() string {
	return "project_views"
}

// Every project and saved filter gets the fork's labels view as its last tab,
// like new ones do in CreateDefaultViewsForProject.
func addLabelsViews20261021120000(tx *xorm.Engine) error {
	projectIDs := []int64{}
	if err := tx.Table("projects").Cols("id").Find(&projectIDs); err != nil {
		return fmt.Errorf("could not get projects: %w", err)
	}

	filterIDs := []int64{}
	if err := tx.Table("saved_filters").Cols("id").Find(&filterIDs); err != nil {
		return fmt.Errorf("could not get saved filters: %w", err)
	}
	for _, id := range filterIDs {
		// Saved filters own their views under project_id -(filter id + 1).
		projectIDs = append(projectIDs, -id-1)
	}

	views := []*projectView20261021120000{}
	if err := tx.Cols("project_id", "view_kind", "position").Find(&views); err != nil {
		return fmt.Errorf("could not get views: %w", err)
	}
	maxPosition := make(map[int64]float64, len(views))
	hasLabels := make(map[int64]bool, len(views))
	for _, v := range views {
		maxPosition[v.ProjectID] = max(maxPosition[v.ProjectID], v.Position)
		if v.ViewKind == viewKindLabels20261021120000 {
			hasLabels[v.ProjectID] = true
		}
	}

	for _, id := range projectIDs {
		if hasLabels[id] {
			continue
		}
		_, err := tx.Insert(&projectView20261021120000{
			Title:     "Labels",
			ProjectID: id,
			ViewKind:  viewKindLabels20261021120000,
			Position:  maxPosition[id] + 100,
		})
		if err != nil {
			return fmt.Errorf("could not add labels view to project %d: %w", id, err)
		}
	}
	return nil
}

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20261021120000",
		Description: "Add a labels view to every project and saved filter.",
		Migrate:     addLabelsViews20261021120000,
		Rollback: func(_ *xorm.Engine) error {
			return nil
		},
	})
}
