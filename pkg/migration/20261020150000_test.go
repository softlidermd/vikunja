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
	"code.vikunja.io/api/pkg/user"

	"github.com/stretchr/testify/require"
)

func TestMakeUsersDiscoverable20261020150000(t *testing.T) {
	x, err := db.CreateTestEngine()
	require.NoError(t, err)
	require.NoError(t, x.Sync2(&user.User{}))

	hidden := &user.User{
		Username: "hidden20261020150000",
		Email:    "hidden20261020150000@example.com",
	}
	_, err = x.Insert(hidden)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := x.ID(hidden.ID).Delete(&user.User{})
		require.NoError(t, err)
	})
	_, err = x.ID(hidden.ID).Cols("discoverable_by_name").Update(&user.User{DiscoverableByName: false})
	require.NoError(t, err)

	require.NoError(t, makeUsersDiscoverable20261020150000(x))

	got := &user.User{}
	has, err := x.ID(hidden.ID).Get(got)
	require.NoError(t, err)
	require.True(t, has)
	require.True(t, got.DiscoverableByName, "existing user should be discoverable by name")
}
