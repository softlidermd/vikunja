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
	"code.vikunja.io/api/pkg/license"
	"code.vikunja.io/api/pkg/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminBypass_Project(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	license.SetForTests([]license.Feature{license.FeatureAdminPanel})
	defer license.ResetForTests()
	s := db.NewSession()
	defer s.Close()

	_, err := s.ID(int64(2)).Cols("is_admin").Update(&user.User{IsAdmin: true})
	require.NoError(t, err)

	admin := &user.User{ID: 2, IsAdmin: true}
	p := &Project{ID: 1}

	t.Run("CanRead", func(t *testing.T) {
		can, _, err := p.CanRead(s, admin)
		require.NoError(t, err)
		assert.True(t, can, "admin must be able to read any project")
	})

	t.Run("CanWrite", func(t *testing.T) {
		can, err := p.CanWrite(s, admin)
		require.NoError(t, err)
		assert.True(t, can)
	})

	t.Run("CanUpdate", func(t *testing.T) {
		can, err := p.CanUpdate(s, admin)
		require.NoError(t, err)
		assert.True(t, can)
	})

	t.Run("CanDelete", func(t *testing.T) {
		can, err := p.CanDelete(s, admin)
		require.NoError(t, err)
		assert.True(t, can)
	})
}

// A stale JWT admin claim must not grant the bypass after DB demotion.
func TestAdminBypass_StaleJWT_Demoted(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	license.SetForTests([]license.Feature{license.FeatureAdminPanel})
	defer license.ResetForTests()
	s := db.NewSession()
	defer s.Close()

	stale := &user.User{ID: 3, IsAdmin: true}
	p := &Project{ID: 1}

	t.Run("CanRead", func(t *testing.T) {
		can, _, err := p.CanRead(s, stale)
		require.NoError(t, err)
		assert.False(t, can, "stale admin claim must not grant project read without DB confirmation")
	})

	t.Run("CanWrite", func(t *testing.T) {
		can, err := p.CanWrite(s, stale)
		require.NoError(t, err)
		assert.False(t, can)
	})

	t.Run("CanDelete", func(t *testing.T) {
		can, err := p.CanDelete(s, stale)
		require.NoError(t, err)
		assert.False(t, can)
	})
}

func TestAdminBypass_StaleJWT_DeletedUser(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	license.SetForTests([]license.Feature{license.FeatureAdminPanel})
	defer license.ResetForTests()
	s := db.NewSession()
	defer s.Close()

	stale := &user.User{ID: 99999, IsAdmin: true}
	p := &Project{ID: 1}

	t.Run("CanRead", func(t *testing.T) {
		can, _, err := p.CanRead(s, stale)
		require.NoError(t, err)
		assert.False(t, can, "deleted admin must not grant bypass")
	})
}

// Label endpoints use only the caller's own memberships: being an instance admin adds nothing.
func TestAdminBypass_Label(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	license.SetForTests([]license.Feature{license.FeatureAdminPanel})
	defer license.ResetForTests()
	s := db.NewSession()
	defer s.Close()

	_, err := s.ID(int64(2)).Cols("is_admin").Update(&user.User{IsAdmin: true})
	require.NoError(t, err)
	admin := &user.User{ID: 2, IsAdmin: true}

	t.Run("project the admin is not a member of", func(t *testing.T) {
		label := &Label{ID: 1}
		can, _, err := label.CanRead(s, admin)
		require.NoError(t, err)
		assert.False(t, can, "CanRead")

		can, err = (&Label{ID: 1}).CanUpdate(s, admin)
		require.NoError(t, err)
		assert.False(t, can, "CanUpdate")

		can, err = (&Label{ID: 1}).CanDelete(s, admin)
		require.NoError(t, err)
		assert.False(t, can, "CanDelete")

		can, err = (&Label{Title: "admin", ProjectID: 1}).CanCreate(s, admin)
		require.NoError(t, err)
		assert.False(t, can, "CanCreate")

		ls, _, _, err := (&Label{}).ReadAll(s, admin, "", 0, 0)
		require.NoError(t, err)
		assert.NotContains(t, labelIDs(t, ls), int64(1))
	})

	t.Run("project the admin can only read", func(t *testing.T) {
		// User 2 has read on project 3 (users_projects.yml).
		label := &Label{
			Title:       "in project 3",
			ProjectID:   3,
			CreatedByID: 3,
		}
		_, err := s.Insert(label)
		require.NoError(t, err)

		can, _, err := (&Label{ID: label.ID}).CanRead(s, admin)
		require.NoError(t, err)
		assert.True(t, can, "CanRead")

		can, err = (&Label{ID: label.ID}).CanUpdate(s, admin)
		require.NoError(t, err)
		assert.False(t, can, "CanUpdate")

		can, err = (&Label{ID: label.ID}).CanDelete(s, admin)
		require.NoError(t, err)
		assert.False(t, can, "CanDelete")

		can, err = (&Label{Title: "admin", ProjectID: 3}).CanCreate(s, admin)
		require.NoError(t, err)
		assert.False(t, can, "CanCreate")
	})
}
