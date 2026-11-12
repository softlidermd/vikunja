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
	"sort"
	"time"

	"code.vikunja.io/api/pkg/log"

	"src.techknowlogick.com/xormigrate"
	"xorm.io/builder"
	"xorm.io/xorm"
)

type labels20261112120000 struct {
	ProjectID    int64 `xorm:"bigint not null default 0 INDEX"`
	ClonedFromID int64 `xorm:"bigint not null default 0 INDEX"`
}

func (labels20261112120000) TableName() string {
	return "labels"
}

// No created/updated tags, so clones keep the original's timestamps.
type labelRow20261112120000 struct {
	ID           int64     `xorm:"bigint autoincr not null unique pk"`
	Title        string    `xorm:"varchar(250) not null"`
	Description  string    `xorm:"longtext null"`
	HexColor     string    `xorm:"varchar(6) null"`
	CreatedByID  int64     `xorm:"bigint not null"`
	ProjectID    int64     `xorm:"bigint not null default 0"`
	ClonedFromID int64     `xorm:"bigint not null default 0"`
	Created      time.Time `xorm:"not null"`
	Updated      time.Time `xorm:"not null"`
}

func (labelRow20261112120000) TableName() string {
	return "labels"
}

type labelTask20261112120000 struct {
	ID      int64 `xorm:"bigint autoincr not null unique pk"`
	TaskID  int64 `xorm:"bigint not null"`
	LabelID int64 `xorm:"bigint not null"`
}

func (labelTask20261112120000) TableName() string {
	return "label_tasks"
}

type labelUsage20261112120000 struct {
	LabelID   int64
	ProjectID int64
	N         int64
}

type labelCreator20261112120000 struct {
	ID         int64
	BotOwnerID int64
}

// Moves every label into exactly one project: a label used in several projects
// stays in its most-used one and is cloned into the others, an unused one goes
// to a project its (bot-resolved) creator owns, or is deleted.
func migrateLabelsToProjects20261112120000(tx *xorm.Engine) error {
	if err := partialSync(tx, labels20261112120000{}); err != nil {
		return err
	}

	sess := tx.NewSession()
	// Close rolls back whatever was not committed.
	defer sess.Close()
	if err := sess.Begin(); err != nil {
		return fmt.Errorf("could not start transaction: %w", err)
	}

	if err := moveLabelsIntoProjects20261112120000(sess); err != nil {
		return err
	}

	if err := sess.Commit(); err != nil {
		return fmt.Errorf("could not commit label migration: %w", err)
	}
	return nil
}

func moveLabelsIntoProjects20261112120000(sess *xorm.Session) error {
	orphanConds := []builder.Cond{
		builder.NotIn("task_id", builder.Select("id").From("tasks")),
		builder.In("task_id", builder.Select("id").From("tasks").Where(
			builder.NotIn("project_id", builder.Select("id").From("projects")),
		)),
		builder.NotIn("label_id", builder.Select("id").From("labels")),
	}
	var orphans int64
	for _, cond := range orphanConds {
		n, err := sess.Where(cond).Delete(&labelTask20261112120000{})
		if err != nil {
			return fmt.Errorf("could not delete orphaned label_tasks: %w", err)
		}
		orphans += n
	}
	log.Infof("Labels to projects: deleted %d orphaned label_tasks rows", orphans)

	labels := []*labelRow20261112120000{}
	if err := sess.Where(builder.Eq{"project_id": 0}).OrderBy("id ASC").Find(&labels); err != nil {
		return fmt.Errorf("could not get labels: %w", err)
	}

	// Soft-deleted tasks count: their labels must survive a restore.
	usage := []*labelUsage20261112120000{}
	err := sess.Table("label_tasks").
		Select("label_tasks.label_id, tasks.project_id, COUNT(*) AS n").
		Join("INNER", "tasks", "tasks.id = label_tasks.task_id").
		GroupBy("label_tasks.label_id, tasks.project_id").
		Find(&usage)
	if err != nil {
		return fmt.Errorf("could not get label usage: %w", err)
	}
	usageByLabel := make(map[int64][]*labelUsage20261112120000, len(usage))
	for _, u := range usage {
		usageByLabel[u.LabelID] = append(usageByLabel[u.LabelID], u)
	}

	var kept, cloned, moved, deleted int
	for _, l := range labels {
		projects := usageByLabel[l.ID]
		if len(projects) == 0 {
			projectID, err := ownedProjectForLabel20261112120000(sess, l.CreatedByID)
			if err != nil {
				return err
			}
			if projectID == 0 {
				if _, err := sess.ID(l.ID).Delete(&labelRow20261112120000{}); err != nil {
					return fmt.Errorf("could not delete label %d: %w", l.ID, err)
				}
				log.Infof("Labels to projects: deleted unreachable label %d %q", l.ID, l.Title)
				deleted++
				continue
			}
			if err := setLabelProject20261112120000(sess, l.ID, projectID); err != nil {
				return err
			}
			moved++
			continue
		}

		// Most rows keep the original id; ties go to the lowest project id.
		sort.Slice(projects, func(i, j int) bool {
			if projects[i].N != projects[j].N {
				return projects[i].N > projects[j].N
			}
			return projects[i].ProjectID < projects[j].ProjectID
		})
		if err := setLabelProject20261112120000(sess, l.ID, projects[0].ProjectID); err != nil {
			return err
		}
		kept++

		for _, p := range projects[1:] {
			clone := &labelRow20261112120000{
				Title:        l.Title,
				Description:  l.Description,
				HexColor:     l.HexColor,
				CreatedByID:  l.CreatedByID,
				ProjectID:    p.ProjectID,
				ClonedFromID: l.ID,
				Created:      l.Created,
				Updated:      l.Updated,
			}
			if _, err := sess.Insert(clone); err != nil {
				return fmt.Errorf("could not clone label %d into project %d: %w", l.ID, p.ProjectID, err)
			}
			_, err := sess.
				Where(builder.And(
					builder.Eq{"label_id": l.ID},
					builder.In("task_id", builder.Select("id").From("tasks").Where(builder.Eq{"project_id": p.ProjectID})),
				)).
				Cols("label_id").
				Update(&labelTask20261112120000{LabelID: clone.ID})
			if err != nil {
				return fmt.Errorf("could not move label %d rows of project %d to clone %d: %w", l.ID, p.ProjectID, clone.ID, err)
			}
			cloned++
		}
	}

	if err := checkLabelsInProjects20261112120000(sess); err != nil {
		return err
	}

	log.Infof("Labels to projects: %d kept, %d cloned, %d moved, %d deleted", kept, cloned, moved, deleted)
	return nil
}

func setLabelProject20261112120000(sess *xorm.Session, labelID, projectID int64) error {
	_, err := sess.ID(labelID).Cols("project_id").Update(&labelRow20261112120000{ProjectID: projectID})
	if err != nil {
		return fmt.Errorf("could not set project of label %d: %w", labelID, err)
	}
	return nil
}

// Returns 0 when the creator (or the bot's owner) owns no project, or no longer exists.
func ownedProjectForLabel20261112120000(sess *xorm.Session, creatorID int64) (int64, error) {
	creator := &labelCreator20261112120000{}
	has, err := sess.Table("users").Select("id, bot_owner_id").Where(builder.Eq{"id": creatorID}).Get(creator)
	if err != nil {
		return 0, fmt.Errorf("could not get user %d: %w", creatorID, err)
	}
	if !has {
		return 0, nil
	}
	human := creator.ID
	if creator.BotOwnerID > 0 {
		human = creator.BotOwnerID
	}

	var projectID int64
	has, err = sess.Table("projects").
		Select("projects.id").
		Join("INNER", "users", "users.default_project_id = projects.id").
		Where(builder.Eq{
			"users.id":          human,
			"projects.owner_id": human,
		}).
		Get(&projectID)
	if err != nil {
		return 0, fmt.Errorf("could not get default project of user %d: %w", human, err)
	}
	if has {
		return projectID, nil
	}

	has, err = sess.Table("projects").
		Select("id").
		Where(builder.Eq{"owner_id": human}).
		OrderBy("is_archived ASC, id ASC").
		Limit(1).
		Get(&projectID)
	if err != nil {
		return 0, fmt.Errorf("could not get a project of user %d: %w", human, err)
	}
	if !has {
		return 0, nil
	}
	return projectID, nil
}

func checkLabelsInProjects20261112120000(sess *xorm.Session) error {
	unplaced, err := sess.Table("labels").Where(builder.Eq{"project_id": 0}).Count()
	if err != nil {
		return fmt.Errorf("could not count labels without project: %w", err)
	}
	if unplaced > 0 {
		return fmt.Errorf("%d labels still have no project", unplaced)
	}

	crossProject, err := sess.Table("label_tasks").
		Join("INNER", "tasks", "tasks.id = label_tasks.task_id").
		Join("INNER", "labels", "labels.id = label_tasks.label_id").
		Where("labels.project_id <> tasks.project_id").
		Count()
	if err != nil {
		return fmt.Errorf("could not count cross-project label_tasks: %w", err)
	}
	if crossProject > 0 {
		return fmt.Errorf("%d label_tasks rows link a label of another project", crossProject)
	}

	dangling, err := sess.Table("label_tasks").
		Join("LEFT", "labels", "labels.id = label_tasks.label_id").
		Where(builder.IsNull{"labels.id"}).
		Count()
	if err != nil {
		return fmt.Errorf("could not count label_tasks without label: %w", err)
	}
	if dangling > 0 {
		return fmt.Errorf("%d label_tasks rows point to a missing label", dangling)
	}
	return nil
}

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20261112120000",
		Description: "Make labels per project: add labels.project_id and labels.cloned_from_id and move every label into a project",
		Migrate:     migrateLabelsToProjects20261112120000,
		Rollback: func(_ *xorm.Engine) error {
			return nil
		},
	})
}
