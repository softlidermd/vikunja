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
	"testing"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/user"
	"code.vikunja.io/api/pkg/web"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/d4l3k/messagediff.v1"
)

// Fixture memberships the label tests rely on:
// user 1 owns 1, 21 and 22 (21 and 22 archived), reads 3, and writes 16 through
// team 1's share on its parent 33. User 13 owns 20, user 21 owns 44, bot 23 is
// admin on 44, user 15 owns 36. User 2 reads 3 and 21 (team 1). Link share 1
// reads project 1.
func labelIDs(t *testing.T, ls interface{}) []int64 {
	t.Helper()
	labels, ok := ls.([]*LabelWithTaskID)
	require.True(t, ok)
	ids := make([]int64, 0, len(labels))
	for _, l := range labels {
		ids = append(ids, l.ID)
	}
	return ids
}

func TestLabel_ReadAll(t *testing.T) {
	t.Run("labels of the member projects", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		gotLs, _, _, err := (&Label{}).ReadAll(s, &user.User{ID: 1}, "", 0, 0)
		require.NoError(t, err)
		// Project 1: 1, 2, 4, 5, 7, 13. Project 16: 10. Project 21: 14, 17. Project 22: 15.
		// Not 6 and 8 (project 20), 9, 11 and 12 (project 44), 16 (project 36).
		assert.Equal(t, []int64{1, 2, 4, 5, 7, 10, 13, 14, 15, 17}, labelIDs(t, gotLs))

		labels := gotLs.([]*LabelWithTaskID)
		assert.Equal(t, int64(1), labels[2].ProjectID)
		require.NotNil(t, labels[2].CreatedBy)
		assert.Equal(t, "user2", labels[2].CreatedBy.Username)
	})
	t.Run("invalid user", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		_, _, _, err := (&Label{}).ReadAll(s, &user.User{ID: -1}, "", 0, 0)
		require.Error(t, err)
	})
}

func TestLabel_ReadAll_LinkShare(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	share, err := GetLinkShareByID(s, 1)
	require.NoError(t, err)

	gotLs, _, _, err := (&Label{}).ReadAll(s, share, "", 0, 0)
	require.NoError(t, err)
	assert.Equal(t, []int64{1, 2, 4, 5, 7, 13}, labelIDs(t, gotLs), "link share for project 1 sees exactly project 1's labels")
}

func TestLabel_ReadAll_SearchMatchesDescription(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	_, err := s.ID(7).Cols("description").Update(&Label{Description: "haystackneedle"})
	require.NoError(t, err)

	gotLs, _, _, err := (&Label{}).ReadAll(s, &user.User{ID: 1}, "haystackneedle", 0, 0)
	require.NoError(t, err)
	assert.Equal(t, []int64{7}, labelIDs(t, gotLs))
}

func TestLabel_ReadOne(t *testing.T) {
	user1 := &user.User{
		ID:                           1,
		Username:                     "user1",
		Password:                     "$2a$04$X4aRMEt0ytgPwMIgv36cI..7X9.nhY/.tYwxpqSi0ykRHx2CwQ0S6",
		Issuer:                       "local",
		EmailRemindersEnabled:        true,
		OverdueTasksRemindersEnabled: true,
		OverdueTasksRemindersTime:    "09:00",
		Created:                      testCreatedTime,
		Updated:                      testUpdatedTime,
		ExportFileID:                 1,
	}
	tests := []struct {
		name              string
		labelID           int64
		auth              web.Auth
		wantForbidden     bool
		wantMaxPermission Permission
		want              *Label
	}{
		{
			name:              "owner of the label's project",
			labelID:           1,
			auth:              &user.User{ID: 1},
			wantMaxPermission: PermissionAdmin,
			want: &Label{
				ID:          1,
				Title:       "Label #1",
				ProjectID:   1,
				CreatedByID: 1,
				CreatedBy:   user1,
				Created:     testCreatedTime,
				Updated:     testUpdatedTime,
			},
		},
		{
			name:              "project member, not the creator",
			labelID:           4,
			auth:              &user.User{ID: 1},
			wantMaxPermission: PermissionAdmin,
		},
		{
			name:              "team share inherited from the parent project",
			labelID:           10,
			auth:              &user.User{ID: 1},
			wantMaxPermission: PermissionWrite,
		},
		{
			name:              "archived project stays readable",
			labelID:           14,
			auth:              &user.User{ID: 1},
			wantMaxPermission: PermissionAdmin,
		},
		{
			// Team 1 reads project 21; user 2 created label 17 but gets nothing extra for it.
			name:              "read-only member",
			labelID:           17,
			auth:              &user.User{ID: 2},
			wantMaxPermission: PermissionRead,
		},
		{
			name:              "label created by a bot, read through project 44",
			labelID:           11,
			auth:              &user.User{ID: 21},
			wantMaxPermission: PermissionAdmin,
		},
		{
			name:              "bot member of the label's project",
			labelID:           9,
			auth:              &user.User{ID: 23},
			wantMaxPermission: PermissionAdmin,
		},
		{
			name:              "link share of the label's project",
			labelID:           1,
			auth:              &LinkSharing{ID: 1, ProjectID: 1, Permission: PermissionRead},
			wantMaxPermission: PermissionRead,
		},
		{
			// GHSA-hj5c-mhh2-g7jq: label 6 lives in user 13's private project 20.
			name:          "non-member",
			labelID:       6,
			auth:          &user.User{ID: 1},
			wantForbidden: true,
		},
		{
			name:          "creator who is not a member of the label's project",
			labelID:       8,
			auth:          &user.User{ID: 1},
			wantForbidden: true,
		},
		{
			name:          "unrelated user",
			labelID:       9,
			auth:          &user.User{ID: 22},
			wantForbidden: true,
		},
		{
			name:          "link share of another project",
			labelID:       6,
			auth:          &LinkSharing{ID: 1, ProjectID: 1, Permission: PermissionRead},
			wantForbidden: true,
		},
		{
			// Reads don't disclose whether a label id exists.
			name:          "nonexistent label",
			labelID:       9999,
			auth:          &user.User{ID: 1},
			wantForbidden: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db.LoadAndAssertFixtures(t)
			s := db.NewSession()
			defer s.Close()

			l := &Label{ID: tt.labelID}
			allowed, maxPermission, err := l.CanRead(s, tt.auth)
			require.NoError(t, err)
			if tt.wantForbidden {
				assert.False(t, allowed)
				return
			}
			require.True(t, allowed)
			assert.Equal(t, int(tt.wantMaxPermission), maxPermission)

			require.NoError(t, l.ReadOne(s, tt.auth))
			if tt.want != nil {
				if diff, equal := messagediff.PrettyDiff(l, tt.want); !equal {
					t.Errorf("Label.ReadOne() = %v, want %v, diff: %v", l, tt.want, diff)
				}
			}
		})
	}
}

func TestLabel_ReadOne_DeletedCreator(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	_, err := s.ID(13).Delete(&user.User{})
	require.NoError(t, err)

	l := &Label{ID: 6}
	require.NoError(t, l.ReadOne(s, &user.User{ID: 13}))
	assert.Equal(t, int64(20), l.ProjectID)
	assert.Nil(t, l.CreatedBy)
}

func TestLabel_Create(t *testing.T) {
	tests := []struct {
		name          string
		projectID     int64
		auth          web.Auth
		wantForbidden bool
		wantErr       func(error) bool
	}{
		{
			name:      "owner",
			projectID: 1,
			auth:      &user.User{ID: 1},
		},
		{
			name:      "write through a team share on the parent project",
			projectID: 16,
			auth:      &user.User{ID: 1},
		},
		{
			name:          "read-only member",
			projectID:     3,
			auth:          &user.User{ID: 1},
			wantForbidden: true,
		},
		{
			name:          "non-member",
			projectID:     20,
			auth:          &user.User{ID: 1},
			wantForbidden: true,
		},
		{
			name:          "link share with write access",
			projectID:     2,
			auth:          &LinkSharing{ID: 2, ProjectID: 2, Permission: PermissionWrite},
			wantForbidden: true,
		},
		{
			name:      "no project",
			projectID: 0,
			auth:      &user.User{ID: 1},
			wantErr:   IsErrProjectDoesNotExist,
		},
		{
			name:      "negative project",
			projectID: -1,
			auth:      &user.User{ID: 1},
			wantErr:   IsErrProjectDoesNotExist,
		},
		{
			name:      "missing project",
			projectID: 99999,
			auth:      &user.User{ID: 1},
			wantErr:   IsErrProjectDoesNotExist,
		},
		{
			name:      "archived project",
			projectID: 22,
			auth:      &user.User{ID: 1},
			wantErr:   IsErrProjectIsArchived,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db.LoadAndAssertFixtures(t)
			s := db.NewSession()
			defer s.Close()

			l := &Label{
				Title:       "Test #1",
				Description: "Lorem Ipsum",
				HexColor:    "ffccff",
				ProjectID:   tt.projectID,
			}
			allowed, err := l.CanCreate(s, tt.auth)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.True(t, tt.wantErr(err), "got %v", err)
				return
			}
			require.NoError(t, err)
			if tt.wantForbidden {
				assert.False(t, allowed)
				return
			}
			require.True(t, allowed)

			require.NoError(t, l.Create(s, tt.auth))
			require.NoError(t, s.Commit())
			db.AssertExists(t, "labels", map[string]interface{}{
				"id":            l.ID,
				"title":         l.Title,
				"description":   l.Description,
				"hex_color":     l.HexColor,
				"project_id":    tt.projectID,
				"created_by_id": tt.auth.GetID(),
			}, false)
		})
	}
}

// Create is reachable without CanCreate (importer, MCP), so it must refuse a missing project itself.
func TestLabel_Create_RequiresProject(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	for _, projectID := range []int64{0, -1} {
		err := (&Label{Title: "no project", ProjectID: projectID}).Create(s, &user.User{ID: 1})
		require.Error(t, err)
		assert.True(t, IsErrProjectDoesNotExist(err), "got %v", err)
	}
}

func TestLabel_Update(t *testing.T) {
	tests := []struct {
		name          string
		labelID       int64
		auth          web.Auth
		wantForbidden bool
		wantErr       func(error) bool
	}{
		{
			name:    "write member who did not create the label",
			labelID: 4,
			auth:    &user.User{ID: 1},
		},
		{
			name:    "write through a team share on the parent project",
			labelID: 10,
			auth:    &user.User{ID: 1},
		},
		{
			name:          "read-only member who created the label",
			labelID:       17,
			auth:          &user.User{ID: 2},
			wantForbidden: true,
		},
		{
			name:          "non-member",
			labelID:       6,
			auth:          &user.User{ID: 1},
			wantForbidden: true,
		},
		{
			name:          "creator who is not a member",
			labelID:       8,
			auth:          &user.User{ID: 1},
			wantForbidden: true,
		},
		{
			name:          "link share with admin access",
			labelID:       1,
			auth:          &LinkSharing{ID: 1, ProjectID: 1, Permission: PermissionAdmin},
			wantForbidden: true,
		},
		{
			name:    "archived project",
			labelID: 14,
			auth:    &user.User{ID: 1},
			wantErr: IsErrProjectIsArchived,
		},
		{
			name:    "nonexistent label",
			labelID: 99999,
			auth:    &user.User{ID: 1},
			wantErr: IsErrLabelDoesNotExist,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db.LoadAndAssertFixtures(t)
			s := db.NewSession()
			defer s.Close()

			// The project_id in the body must be ignored: labels never move.
			l := &Label{
				ID:        tt.labelID,
				Title:     "new and better",
				ProjectID: 20,
			}
			allowed, err := l.CanUpdate(s, tt.auth)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.True(t, tt.wantErr(err), "got %v", err)
				return
			}
			require.NoError(t, err)
			if tt.wantForbidden {
				assert.False(t, allowed)
				return
			}
			require.True(t, allowed)

			stored, err := getLabelByIDSimple(s, tt.labelID)
			require.NoError(t, err)

			require.NoError(t, l.Update(s, tt.auth))
			require.NoError(t, s.Commit())
			db.AssertExists(t, "labels", map[string]interface{}{
				"id":         tt.labelID,
				"title":      "new and better",
				"project_id": stored.ProjectID,
			}, false)
		})
	}
}

func TestLabel_Delete(t *testing.T) {
	tests := []struct {
		name          string
		labelID       int64
		auth          web.Auth
		wantForbidden bool
		wantErr       func(error) bool
	}{
		{
			name:    "write member who did not create the label",
			labelID: 4,
			auth:    &user.User{ID: 1},
		},
		{
			name:          "read-only member who created the label",
			labelID:       17,
			auth:          &user.User{ID: 2},
			wantForbidden: true,
		},
		{
			name:          "non-member",
			labelID:       6,
			auth:          &user.User{ID: 1},
			wantForbidden: true,
		},
		{
			name:    "nonexistent label",
			labelID: 99999,
			auth:    &user.User{ID: 1},
			wantErr: IsErrLabelDoesNotExist,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db.LoadAndAssertFixtures(t)
			s := db.NewSession()
			defer s.Close()

			l := &Label{ID: tt.labelID}
			allowed, err := l.CanDelete(s, tt.auth)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.True(t, tt.wantErr(err), "got %v", err)
				return
			}
			require.NoError(t, err)
			if tt.wantForbidden {
				assert.False(t, allowed)
				return
			}
			require.True(t, allowed)

			require.NoError(t, l.Delete(s, tt.auth))
			require.NoError(t, s.Commit())
			db.AssertMissing(t, "labels", map[string]interface{}{
				"id": tt.labelID,
			})
		})
	}
}

// The list and the single-label read must agree; they drifted apart before,
// letting GET /labels/{id} return a label the list hid.
func TestLabel_ReadAllMatchesCanRead(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	share, err := GetLinkShareByID(s, 1)
	require.NoError(t, err)

	var allLabels []*Label
	require.NoError(t, s.Find(&allLabels))
	require.NotEmpty(t, allLabels)

	auths := map[string]web.Auth{
		"user 1":       &user.User{ID: 1},
		"user 13":      &user.User{ID: 13},
		"user 15":      &user.User{ID: 15},
		"user 21":      &user.User{ID: 21},
		"link share 1": share,
	}

	// Pinned from labels.yml and the project memberships, independently of
	// labelVisibleCond, so a widening of the shared cond can't slip through just
	// because both sides of the parity check moved together.
	want := map[string][]int64{
		"user 1":       {1, 2, 4, 5, 7, 10, 13, 14, 15, 17},
		"user 13":      {6, 8},
		"user 15":      {16},
		"user 21":      {9, 11, 12},
		"link share 1": {1, 2, 4, 5, 7, 13},
	}

	for name, a := range auths {
		t.Run(name, func(t *testing.T) {
			gotLs, _, _, err := (&Label{}).ReadAll(s, a, "", 0, 0)
			require.NoError(t, err)
			listed := labelIDs(t, gotLs)

			readable := []int64{}
			for _, lb := range allLabels {
				can, _, err := (&Label{ID: lb.ID}).CanRead(s, a)
				require.NoError(t, err)
				if can {
					readable = append(readable, lb.ID)
				}
			}

			assert.ElementsMatch(t, readable, listed, "ReadAll returned %v but CanRead allows %v", listed, readable)
			assert.ElementsMatch(t, want[name], listed, "ReadAll for %s = %v, want %v", name, listed, want[name])
		})
	}
}

// A cond that came back invalid would be dropped by the callers' builder.And,
// widening their queries to every label instead of erroring.
func TestLabelVisibleCondIsValid(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	share, err := GetLinkShareByID(s, 1)
	require.NoError(t, err)

	auths := map[string]web.Auth{
		"user":                     &user.User{ID: 1},
		"link share":               share,
		"user without any project": &user.User{ID: 17},
	}

	for name, a := range auths {
		t.Run(name, func(t *testing.T) {
			cond, err := labelVisibleCond(s, a)
			require.NoError(t, err)
			require.NotNil(t, cond)
			assert.True(t, cond.IsValid())
		})
	}
}

// Regression for #4026: label_tasks has no soft-delete column, so a deleted
// label used to leave its task associations behind forever.
func TestLabel_DeleteRemovesTaskAssociations(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	// Label 8 is attached to task 34.
	l := &Label{ID: 8}
	require.NoError(t, l.Delete(s, &user.User{ID: 13}))
	require.NoError(t, s.Commit())

	db.AssertMissing(t, "labels", map[string]interface{}{"id": 8})
	db.AssertMissing(t, "label_tasks", map[string]interface{}{"label_id": 8})
	db.AssertExists(t, "label_tasks", map[string]interface{}{"label_id": 4, "task_id": 1}, false)
}

func TestLabel_GetOrCreateProjectLabel(t *testing.T) {
	t.Run("a label of the same family", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		src, err := getLabelByIDSimple(s, 4)
		require.NoError(t, err)
		got, err := GetOrCreateProjectLabel(s, 21, src)
		require.NoError(t, err)
		assert.Equal(t, int64(14), got.ID)

		// From a copy, the root itself is family too.
		src, err = getLabelByIDSimple(s, 15)
		require.NoError(t, err)
		got, err = GetOrCreateProjectLabel(s, 1, src)
		require.NoError(t, err)
		assert.Equal(t, int64(4), got.ID)
	})
	t.Run("an unrelated label with the exact title", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		got, err := GetOrCreateProjectLabel(s, 21, &Label{Title: "Label #5"})
		require.NoError(t, err)
		assert.Equal(t, int64(17), got.ID)

		// Exact: a title differing only in case is not a match.
		got, err = GetOrCreateProjectLabel(s, 21, &Label{Title: "label #5", CreatedByID: 1})
		require.NoError(t, err)
		assert.NotEqual(t, int64(17), got.ID)
	})
	t.Run("a new copy keeping creator and lineage", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		src, err := getLabelByIDSimple(s, 14)
		require.NoError(t, err)
		_, err = s.ID(src.ID).Cols("hex_color").Update(&Label{HexColor: "abcdef"})
		require.NoError(t, err)
		src.HexColor = "#abcdef"
		srcBefore := *src

		got, err := GetOrCreateProjectLabel(s, 20, src)
		require.NoError(t, err)
		assert.Equal(t, srcBefore, *src, "src must not be modified")
		assert.NotZero(t, got.ID)
		require.NoError(t, s.Commit())
		db.AssertExists(t, "labels", map[string]interface{}{
			"id":             got.ID,
			"title":          "Label #4 - visible via other task",
			"hex_color":      "abcdef",
			"project_id":     20,
			"created_by_id":  2,
			"cloned_from_id": 4,
		}, false)
	})
}
