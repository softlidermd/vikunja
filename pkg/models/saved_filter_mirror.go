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
	"fmt"
	"slices"
	"strings"

	"code.vikunja.io/api/pkg/web"

	"xorm.io/builder"
	"xorm.io/xorm"
)

// A manual saved-filter Kanban mirrors each task's project status: a card whose project status
// bucket title equals a filter column title is shown in that column.
func isMirroredFilterView(view *ProjectView) bool {
	return view != nil &&
		GetSavedFilterIDFromProjectID(view.ProjectID) > 0 &&
		view.ViewKind == ProjectViewKindKanban &&
		view.BucketConfigurationMode == BucketConfigurationModeManual
}

func normalizeBucketTitle(title string) string {
	return strings.ToLower(strings.TrimSpace(title))
}

type mirrorMove struct {
	taskID int64
	from   int64
	to     int64
}

// This mirrors the frontend's getStatusKanbanView: a project has a status view only when it
// has exactly one manual Kanban view.
func getStatusViewIDsByProject(s *xorm.Session, projectIDs []int64) (statusViewIDs map[int64]int64, err error) {
	statusViewIDs = map[int64]int64{}
	ambiguous := map[int64]bool{}
	for chunk := range slices.Chunk(projectIDs, idChunkSize) {
		views := []*ProjectView{}
		err = s.
			Cols("id", "project_id").
			Where(builder.And(
				builder.Eq{"view_kind": ProjectViewKindKanban},
				builder.Eq{"bucket_configuration_mode": BucketConfigurationModeManual},
				builder.In("project_id", chunk),
			)).
			Find(&views)
		if err != nil {
			return nil, fmt.Errorf("could not load status views: %w", err)
		}
		for _, v := range views {
			if _, has := statusViewIDs[v.ProjectID]; has {
				ambiguous[v.ProjectID] = true
			}
			statusViewIDs[v.ProjectID] = v.ID
		}
	}
	for projectID := range ambiguous {
		delete(statusViewIDs, projectID)
	}
	return statusViewIDs, nil
}

// Duplicate titles resolve to the first bucket by position, id.
func getFirstBucketIDsByTitle(s *xorm.Session, viewIDs []int64) (bucketIDs map[int64]map[string]int64, err error) {
	bucketIDs = map[int64]map[string]int64{}
	for chunk := range slices.Chunk(viewIDs, idChunkSize) {
		buckets := []*Bucket{}
		err = s.
			Cols("id", "title", "project_view_id").
			In("project_view_id", chunk).
			OrderBy("position asc, id asc").
			Find(&buckets)
		if err != nil {
			return nil, fmt.Errorf("could not load buckets by title: %w", err)
		}
		for _, b := range buckets {
			if bucketIDs[b.ProjectViewID] == nil {
				bucketIDs[b.ProjectViewID] = map[string]int64{}
			}
			title := normalizeBucketTitle(b.Title)
			if _, has := bucketIDs[b.ProjectViewID][title]; !has {
				bucketIDs[b.ProjectViewID][title] = b.ID
			}
		}
	}
	return bucketIDs, nil
}

// loadMirrorTargets maps each (filter view, task) to the column matching the task's project status title.
func loadMirrorTargets(s *xorm.Session, tasks []*Task, viewIDs []int64) (targets map[viewTask]int64, err error) {
	targets = map[viewTask]int64{}
	if len(tasks) == 0 || len(viewIDs) == 0 {
		return targets, nil
	}

	titleByTask, err := loadStatusTitles(s, tasks)
	if err != nil {
		return nil, err
	}
	if len(titleByTask) == 0 {
		return targets, nil
	}

	columns, err := getFirstBucketIDsByTitle(s, viewIDs)
	if err != nil {
		return nil, err
	}

	for _, viewID := range viewIDs {
		for taskID, title := range titleByTask {
			if columnID, has := columns[viewID][normalizeBucketTitle(title)]; has {
				targets[viewTask{viewID: viewID, taskID: taskID}] = columnID
			}
		}
	}
	return targets, nil
}

// loadStatusTitles maps each task to the title of its bucket in its project's status view.
// Tasks of projects without a status view are absent.
func loadStatusTitles(s *xorm.Session, tasks []*Task) (titleByTask map[int64]string, err error) {
	titleByTask = map[int64]string{}
	projectIDs := []int64{}
	for _, t := range tasks {
		if t.ProjectID > 0 {
			projectIDs = append(projectIDs, t.ProjectID)
		}
	}
	slices.Sort(projectIDs)
	projectIDs = slices.Compact(projectIDs)

	statusViewIDs, err := getStatusViewIDsByProject(s, projectIDs)
	if err != nil || len(statusViewIDs) == 0 {
		return titleByTask, err
	}

	type statusTitle struct {
		TaskID        int64
		ProjectViewID int64
		Title         string
	}
	statusViewByTask := map[int64]int64{}
	for chunk := range slices.Chunk(tasks, idChunkSize) {
		taskIDs := []int64{}
		chunkViewIDs := []int64{}
		for _, t := range chunk {
			if viewID, has := statusViewIDs[t.ProjectID]; has {
				statusViewByTask[t.ID] = viewID
				taskIDs = append(taskIDs, t.ID)
				chunkViewIDs = append(chunkViewIDs, viewID)
			}
		}
		if len(taskIDs) == 0 {
			continue
		}
		slices.Sort(chunkViewIDs)
		chunkViewIDs = slices.Compact(chunkViewIDs)

		rows := []*statusTitle{}
		err = s.
			Table("task_buckets").
			Select("task_buckets.task_id, task_buckets.project_view_id, buckets.title").
			Join("INNER", "buckets", "buckets.id = task_buckets.bucket_id").
			Where(builder.And(
				builder.In("task_buckets.task_id", taskIDs),
				builder.In("task_buckets.project_view_id", chunkViewIDs),
			)).
			Find(&rows)
		if err != nil {
			return nil, fmt.Errorf("could not load status titles: %w", err)
		}
		for _, r := range rows {
			// A task can keep rows in another project's view; only its own status view counts.
			if statusViewByTask[r.TaskID] == r.ProjectViewID {
				titleByTask[r.TaskID] = r.Title
			}
		}
	}
	return titleByTask, nil
}

// findStatusBucketByTitle returns the bucket in the project's status view with the given title, or zeros.
func findStatusBucketByTitle(s *xorm.Session, projectID int64, title string) (statusViewID, bucketID int64, err error) {
	statusViewIDs, err := getStatusViewIDsByProject(s, []int64{projectID})
	if err != nil {
		return 0, 0, err
	}
	statusViewID, has := statusViewIDs[projectID]
	if !has {
		return 0, 0, nil
	}

	buckets, err := getFirstBucketIDsByTitle(s, []int64{statusViewID})
	if err != nil {
		return 0, 0, err
	}
	bucketID, has = buckets[statusViewID][normalizeBucketTitle(title)]
	if !has {
		return 0, 0, nil
	}
	return statusViewID, bucketID, nil
}

// applyMirrorMoves moves filter rows compare-and-set: a row changed since it was read keeps its new value.
func applyMirrorMoves(s *xorm.Session, viewID int64, moves []mirrorMove) (err error) {
	type fromTo struct {
		from int64
		to   int64
	}
	taskIDsByMove := map[fromTo][]int64{}
	for _, m := range moves {
		key := fromTo{from: m.from, to: m.to}
		taskIDsByMove[key] = append(taskIDsByMove[key], m.taskID)
	}

	for key, taskIDs := range taskIDsByMove {
		for chunk := range slices.Chunk(taskIDs, idChunkSize) {
			_, err = s.
				Table(&TaskBucket{}).
				Where("project_view_id = ? AND bucket_id = ?", viewID, key.from).
				In("task_id", chunk).
				Update(map[string]any{"bucket_id": key.to})
			if err != nil {
				return fmt.Errorf("could not move mirrored tasks: %w", err)
			}
		}
	}
	return nil
}

// syncMirroredFilterView moves matched cards of a manual saved-filter Kanban into their status column
// before the board is read.
// It runs on the read session without a lock: the compare-and-set update never overwrites a concurrent
// filter-row change, and a target made stale by a concurrent status change is repaired by the
// TaskUpdated listener or the next board read. This is not serializable, and that is accepted.
func syncMirroredFilterView(s *xorm.Session, projects []*Project, view *ProjectView, opts *taskSearchOptions) (err error) {
	projectIDs, _ := getProjectIDsFromProjects(projects)
	if len(projectIDs) == 0 {
		return nil
	}

	// A fresh parse: convertFiltersToDBFilterCond mutates opts.parsedFilters in place.
	filterCond, joinTaskBuckets, err := parseFilterCond(opts.filter, opts.filterTimezone, opts.filterIncludeNulls)
	if err != nil || joinTaskBuckets {
		return err
	}

	type filterRow struct {
		ID        int64
		ProjectID int64
		BucketID  int64
	}
	rows := []*filterRow{}
	err = s.
		Table("tasks").
		Select("tasks.id, tasks.project_id, task_buckets.bucket_id").
		Join("INNER", "task_buckets", "task_buckets.task_id = tasks.id AND task_buckets.project_view_id = ?", view.ID).
		Where(builder.And(
			builder.In("tasks.project_id", projectIDs),
			filterCond,
			taskNotDeletedCond("tasks"),
		)).
		Find(&rows)
	if err != nil {
		return fmt.Errorf("could not load filter rows: %w", err)
	}
	if len(rows) == 0 {
		return nil
	}

	tasks := make([]*Task, 0, len(rows))
	for _, r := range rows {
		tasks = append(tasks, &Task{
			ID:        r.ID,
			ProjectID: r.ProjectID,
		})
	}
	targets, err := loadMirrorTargets(s, tasks, []int64{view.ID})
	if err != nil {
		return err
	}

	moves := []mirrorMove{}
	for _, r := range rows {
		to, has := targets[viewTask{viewID: view.ID, taskID: r.ID}]
		if has && to != r.BucketID {
			moves = append(moves, mirrorMove{
				taskID: r.ID,
				from:   r.BucketID,
				to:     to,
			})
		}
	}
	return applyMirrorMoves(s, view.ID, moves)
}

// moveTaskViaStatusView forwards a drop into a column whose title exists on the task's status view
// to that project bucket, then places the filter row in the column matching where the task landed.
func moveTaskViaStatusView(s *xorm.Session, a web.Auth, b *TaskBucket, view *ProjectView, filterBucket *Bucket, task *Task, oldBucketID int64) (handled bool, err error) {
	statusViewID, projectBucketID, err := findStatusBucketByTitle(s, task.ProjectID, filterBucket.Title)
	if err != nil || projectBucketID == 0 {
		return false, err
	}

	pb := &TaskBucket{
		TaskID:        task.ID,
		BucketID:      projectBucketID,
		ProjectViewID: statusViewID,
		ProjectID:     task.ProjectID,
	}
	// Recursion ends here: a project view is never mirrored.
	err = updateTaskBucket(s, a, pb)
	if err != nil {
		return false, err
	}

	// pb.Bucket is the requested bucket after a repeat reroute and nil on a no-op, so map back from pb.BucketID.
	landedBucket, err := getBucketByID(s, pb.BucketID)
	if err != nil {
		return false, err
	}
	columns, err := getFirstBucketIDsByTitle(s, []int64{view.ID})
	if err != nil {
		return false, err
	}
	landingID, has := columns[view.ID][normalizeBucketTitle(landedBucket.Title)]
	if !has {
		landingID = oldBucketID
		if landingID == 0 {
			landingID, err = getDefaultBucketID(s, view)
			if err != nil {
				return false, err
			}
		}
	}

	filterRowChanged := landingID != oldBucketID
	if filterRowChanged {
		err = (&TaskBucket{
			TaskID:        task.ID,
			ProjectViewID: view.ID,
			BucketID:      landingID,
		}).upsert(s)
		if err != nil {
			return false, err
		}
	}

	// Only the drop column's limit was checked: a landing column differing from it (repeat reroute,
	// duplicate titles) is not limit-checked. Accepted.
	if filterBucket.ID == oldBucketID {
		// The outer limit check is skipped for a same-column drop, so count here to never report 0.
		filterBucket.Count, err = checkBucketLimit(s, a, task, filterBucket, view, 0)
		if err != nil && !IsErrBucketLimitExceeded(err) {
			return false, err
		}
	} else if landingID == filterBucket.ID {
		filterBucket.Count++
	}

	b.BucketID = landingID
	b.Bucket = filterBucket
	switch {
	case pb.Task != nil:
		b.Task = pb.Task
	case filterRowChanged:
		b.Task = task
	default:
		b.Task = nil
	}
	return true, nil
}
