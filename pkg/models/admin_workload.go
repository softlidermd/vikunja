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
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"

	"code.vikunja.io/api/pkg/user"

	"xorm.io/builder"
	"xorm.io/xorm"
)

type WorkloadTask struct {
	ID           int64     `json:"id" doc:"The task id."`
	ProjectID    int64     `json:"project_id" doc:"The id of the task's project."`
	Title        string    `json:"title" doc:"The task title."`
	ProjectTitle string    `json:"project_title" doc:"The title of the task's project."`
	Status       string    `xorm:"-" json:"status" doc:"The title of the task's bucket in its project's status board (the single manual Kanban view). Empty when the project has no status board."`
	Priority     int64     `json:"priority" doc:"The task priority."`
	Size         int64     `json:"size" doc:"Estimated effort in hours: 0 = not estimated, 1, 4, 8, 20 or 40. 999 = long-running."`
	DueDate      time.Time `xorm:"DATETIME null 'due_date'" json:"due_date" doc:"The task due date; zero when unset."`
}

type WorkloadTotals struct {
	Hours       int64 `json:"hours" doc:"Sum of the sizes 1, 4, 8, 20 and 40. Unestimated and long-running tasks are not summed."`
	Unestimated int   `json:"unestimated" doc:"Number of tasks with size 0."`
	Long        int   `json:"long" doc:"Number of long-running tasks (size 999)."`
	TaskCount   int   `json:"task_count" doc:"Number of open tasks that are not ideas."`
	Ideas       int   `json:"ideas" doc:"Number of open tasks in the Idea status."`
}

type WorkloadPerson struct {
	User *user.User `json:"user" doc:"The assignee; null for the unassigned block."`
	WorkloadTotals
	Tasks []*WorkloadTask `json:"tasks" doc:"Open non-idea tasks, ordered by due date (unset last), priority descending, then id."`
}

type Workload struct {
	Total      WorkloadTotals    `json:"total" doc:"Instance-wide totals; each task counts once, whatever its number of assignees."`
	People     []*WorkloadPerson `json:"people" doc:"One block per user with at least one open non-idea task, ordered by display name."`
	Unassigned *WorkloadPerson   `json:"unassigned" doc:"Open non-idea tasks without an assignee, plus the count of unassigned ideas."`
}

func (w *WorkloadTotals) add(t *WorkloadTask) {
	w.TaskCount++
	switch t.Size {
	case 0:
		w.Unestimated++
	case TaskSizeLong:
		w.Long++
	default:
		w.Hours += t.Size
	}
}

// GetWorkload returns the open tasks (not done, not deleted, in a non-archived project) grouped per assignee.
// Tasks in the Idea status are only counted. Not paginated: it aggregates over every open task.
func GetWorkload(s *xorm.Session) (*Workload, error) {
	openCond := builder.And(
		builder.Eq{"tasks.done": false},
		builder.Eq{"projects.is_archived": false},
		taskNotDeletedCond("tasks"),
	)

	tasks := []*WorkloadTask{}
	err := s.
		Table("tasks").
		Select("tasks.id, tasks.project_id, tasks.title, projects.title AS project_title, tasks.priority, tasks.size, tasks.due_date").
		Join("INNER", "projects", "projects.id = tasks.project_id").
		Where(openCond).
		Find(&tasks)
	if err != nil {
		return nil, fmt.Errorf("could not load open tasks: %w", err)
	}

	type assignee struct {
		TaskID int64
		UserID int64
	}
	assignees := []*assignee{}
	err = s.
		Table("task_assignees").
		Select("task_assignees.task_id, task_assignees.user_id").
		Join("INNER", "tasks", "tasks.id = task_assignees.task_id").
		Join("INNER", "projects", "projects.id = tasks.project_id").
		Where(openCond).
		Find(&assignees)
	if err != nil {
		return nil, fmt.Errorf("could not load assignees: %w", err)
	}
	userIDsByTask := map[int64][]int64{}
	userIDs := []int64{}
	for _, a := range assignees {
		userIDsByTask[a.TaskID] = append(userIDsByTask[a.TaskID], a.UserID)
		userIDs = append(userIDs, a.UserID)
	}
	users, err := user.GetUsersByIDs(s, userIDs)
	if err != nil {
		return nil, fmt.Errorf("could not load assigned users: %w", err)
	}

	statusTasks := make([]*Task, 0, len(tasks))
	for _, t := range tasks {
		statusTasks = append(statusTasks, &Task{
			ID:        t.ID,
			ProjectID: t.ProjectID,
		})
	}
	titleByTask, err := loadStatusTitles(s, statusTasks)
	if err != nil {
		return nil, fmt.Errorf("could not load task statuses: %w", err)
	}

	slices.SortFunc(tasks, func(a, b *WorkloadTask) int {
		if a.DueDate.IsZero() != b.DueDate.IsZero() {
			if a.DueDate.IsZero() {
				return 1
			}
			return -1
		}
		return cmp.Or(a.DueDate.Compare(b.DueDate), cmp.Compare(b.Priority, a.Priority), cmp.Compare(a.ID, b.ID))
	})

	w := &Workload{Unassigned: &WorkloadPerson{Tasks: []*WorkloadTask{}}}
	people := map[int64]*WorkloadPerson{}
	for _, t := range tasks {
		t.Status = titleByTask[t.ID]
		var blocks []*WorkloadPerson
		for _, id := range userIDsByTask[t.ID] {
			u, has := users[id]
			if !has {
				continue
			}
			if people[id] == nil {
				people[id] = &WorkloadPerson{
					User:  u,
					Tasks: []*WorkloadTask{},
				}
			}
			blocks = append(blocks, people[id])
		}
		if len(blocks) == 0 {
			blocks = []*WorkloadPerson{w.Unassigned}
		}

		if normalizeBucketTitle(t.Status) == "idea" {
			w.Total.Ideas++
			for _, b := range blocks {
				b.Ideas++
			}
			continue
		}
		w.Total.add(t)
		for _, b := range blocks {
			b.add(t)
			b.Tasks = append(b.Tasks, t)
		}
	}

	w.People = []*WorkloadPerson{}
	for _, p := range people {
		// Idea-only assignees get no block; their ideas are in Total.Ideas.
		if len(p.Tasks) > 0 {
			w.People = append(w.People, p)
		}
	}
	slices.SortFunc(w.People, func(a, b *WorkloadPerson) int {
		return cmp.Or(strings.Compare(strings.ToLower(a.User.GetName()), strings.ToLower(b.User.GetName())), cmp.Compare(a.User.ID, b.User.ID))
	})
	return w, nil
}
