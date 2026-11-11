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
	"strings"
	"time"

	"code.vikunja.io/api/pkg/log"

	"src.techknowlogick.com/xormigrate"
	"xorm.io/builder"
	"xorm.io/xorm"
)

type projectView20261111120000 struct {
	ID                      int64 `xorm:"autoincr not null unique pk"`
	ProjectID               int64 `xorm:"not null index"`
	ViewKind                int   `xorm:"not null"`
	BucketConfigurationMode int   `xorm:"default 0"`
	DefaultBucketID         int64 `xorm:"bigint INDEX null"`
	DoneBucketID            int64 `xorm:"bigint INDEX null"`
}

func (projectView20261111120000) TableName() string {
	return "project_views"
}

type bucket20261111120000 struct {
	ID            int64     `xorm:"bigint autoincr not null unique pk"`
	Title         string    `xorm:"text not null"`
	ProjectViewID int64     `xorm:"bigint not null"`
	Position      float64   `xorm:"double null"`
	CreatedByID   int64     `xorm:"bigint not null"`
	Created       time.Time `xorm:"created not null"`
	Updated       time.Time `xorm:"updated not null"`
}

func (bucket20261111120000) TableName() string {
	return "buckets"
}

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20261111120000",
		Description: "Add an Idea bucket to every project status board (as its default) and every manual filter board",
		Migrate:     addIdeaBuckets20261111120000,
		Rollback: func(_ *xorm.Engine) error {
			return nil
		},
	})
}

// Status boards (the single manual Kanban of a project) get Idea leftmost as their default, like new boards.
// Manual filter boards get a non-default Idea column. Tasks keep their buckets. Safe to re-run.
func addIdeaBuckets20261111120000(tx *xorm.Engine) error {
	views := []*projectView20261111120000{}
	// 3 = kanban, 1 = manual buckets. Filter boards live under project_id -(filter id + 1); -1 is Favorites.
	err := tx.
		Where(builder.Eq{"view_kind": 3, "bucket_configuration_mode": 1}).
		And(builder.Or(builder.Gt{"project_id": 0}, builder.Lt{"project_id": -1})).
		OrderBy("id").
		Find(&views)
	if err != nil {
		return fmt.Errorf("could not get manual kanban views: %w", err)
	}

	boardsPerProject := map[int64]int{}
	for _, v := range views {
		if v.ProjectID > 0 {
			boardsPerProject[v.ProjectID]++
		}
	}
	for projectID, count := range boardsPerProject {
		if count > 1 {
			log.Infof("Skipping project %d: %d manual kanban views", projectID, count)
		}
	}

	for _, view := range views {
		if boardsPerProject[view.ProjectID] > 1 {
			continue
		}
		if err := addIdeaBucket20261111120000(tx, view); err != nil {
			return err
		}
	}
	return nil
}

func addIdeaBucket20261111120000(tx *xorm.Engine, view *projectView20261111120000) error {
	buckets := []*bucket20261111120000{}
	if err := tx.Where(builder.Eq{"project_view_id": view.ID}).OrderBy("position, id").Find(&buckets); err != nil {
		return fmt.Errorf("could not get the buckets of view %d: %w", view.ID, err)
	}
	if len(buckets) == 0 {
		log.Infof("Skipping view %d of project %d: it has no buckets", view.ID, view.ProjectID)
		return nil
	}

	isStatusBoard := view.ProjectID > 0
	var idea *bucket20261111120000
	for _, b := range buckets {
		if strings.ToLower(strings.TrimSpace(b.Title)) == "idea" {
			idea = b
			break
		}
	}

	if idea == nil {
		ownerID, found, err := ownerID20261111120000(tx, view.ProjectID)
		if err != nil {
			return err
		}
		// No owner means the project or filter is gone; an insert would dangle created_by_id at 0.
		if !found {
			log.Infof("Skipping view %d: project %d has no owner", view.ID, view.ProjectID)
			return nil
		}

		// A filter board without an explicit default uses its leftmost bucket. Pin that one first, or the new
		// leftmost Idea would silently become the default for unmatched tasks.
		if !isStatusBoard && view.DefaultBucketID == 0 {
			view.DefaultBucketID = buckets[0].ID
			if err := updateDefaultBucket20261111120000(tx, view); err != nil {
				return err
			}
		}

		minPosition := buckets[0].Position
		for _, b := range buckets {
			minPosition = min(minPosition, b.Position)
		}
		position := minPosition - 1
		if minPosition > 0 {
			position = minPosition / 2
		}
		idea = &bucket20261111120000{Title: "Idea", ProjectViewID: view.ID, Position: position, CreatedByID: ownerID}
		if _, err := tx.Insert(idea); err != nil {
			return fmt.Errorf("could not add the Idea bucket to view %d: %w", view.ID, err)
		}
	}

	if !isStatusBoard || view.DefaultBucketID == idea.ID {
		return nil
	}
	view.DefaultBucketID = idea.ID
	return updateDefaultBucket20261111120000(tx, view)
}

func updateDefaultBucket20261111120000(tx *xorm.Engine, view *projectView20261111120000) error {
	if _, err := tx.Where(builder.Eq{"id": view.ID}).Cols("default_bucket_id").Update(view); err != nil {
		return fmt.Errorf("could not set the default bucket of view %d: %w", view.ID, err)
	}
	return nil
}

func ownerID20261111120000(tx *xorm.Engine, projectID int64) (ownerID int64, found bool, err error) {
	if projectID > 0 {
		found, err = tx.Table("projects").Where(builder.Eq{"id": projectID}).Cols("owner_id").Get(&ownerID)
	} else {
		found, err = tx.Table("saved_filters").Where(builder.Eq{"id": -projectID - 1}).Cols("owner_id").Get(&ownerID)
	}
	if err != nil {
		return 0, false, fmt.Errorf("could not get the owner of project %d: %w", projectID, err)
	}
	return ownerID, found, nil
}
