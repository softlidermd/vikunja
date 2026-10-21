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

type projectView20261021120000 struct {
	ID                      int64     `xorm:"autoincr not null unique pk"`
	Title                   string    `xorm:"varchar(255) not null"`
	ProjectID               int64     `xorm:"not null index"`
	ViewKind                int       `xorm:"not null"`
	Filter                  *string   `xorm:"json null default null"`
	Position                float64   `xorm:"double null"`
	BucketConfigurationMode int       `xorm:"default 0"`
	BucketConfiguration     []any     `xorm:"json"`
	DefaultBucketID         int64     `xorm:"bigint INDEX null"`
	DoneBucketID            int64     `xorm:"bigint INDEX null"`
	Updated                 time.Time `xorm:"updated not null"`
	Created                 time.Time `xorm:"created not null"`
}

func (projectView20261021120000) TableName() string {
	return "project_views"
}

type viewMaxPosition20261021120000 struct {
	ProjectID   int64   `xorm:"'project_id'"`
	MaxPosition float64 `xorm:"'max_position'"`
}

// Every project and saved filter gets the fork's labels view as its last tab,
// like new ones do in CreateDefaultViewsForProject.
func addLabelsViews20261021120000(tx *xorm.Engine) error {
	rows := []*viewMaxPosition20261021120000{}
	err := tx.Table("project_views").
		Select("project_id, COALESCE(MAX(position), 0) AS max_position").
		// Saved filters own views under project_id -(filter id + 1); views left behind by deleted ones are skipped.
		Where("project_id IN (SELECT id FROM projects) OR (-project_id - 1) IN (SELECT id FROM saved_filters)").
		GroupBy("project_id").
		Find(&rows)
	if err != nil {
		return fmt.Errorf("could not get view positions: %w", err)
	}

	existing := []int64{}
	err = tx.Table("project_views").Where("view_kind = ?", viewKindLabels20261021120000).Cols("project_id").Find(&existing)
	if err != nil {
		return fmt.Errorf("could not get existing labels views: %w", err)
	}
	has := make(map[int64]bool, len(existing))
	for _, id := range existing {
		has[id] = true
	}

	for _, row := range rows {
		if has[row.ProjectID] {
			continue
		}
		_, err = tx.Insert(&projectView20261021120000{
			Title:     "Labels",
			ProjectID: row.ProjectID,
			ViewKind:  viewKindLabels20261021120000,
			Position:  row.MaxPosition + 100,
		})
		if err != nil {
			return fmt.Errorf("could not add labels view to project %d: %w", row.ProjectID, err)
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
