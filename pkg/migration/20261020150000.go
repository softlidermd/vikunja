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

	"src.techknowlogick.com/xormigrate"
	"xorm.io/builder"
	"xorm.io/xorm"
)

// Accounts made before this fork defaulted discoverable_by_name to true were created hidden,
// so the user pickers that list colleagues on click came up empty for them.
func makeUsersDiscoverable20261020150000(tx *xorm.Engine) error {
	if _, err := tx.Table("users").Where(builder.Eq{"discoverable_by_name": false}).Update(map[string]any{"discoverable_by_name": true}); err != nil {
		return fmt.Errorf("could not make users discoverable by name: %w", err)
	}
	return nil
}

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20261020150000",
		Description: "Make existing users discoverable by name, matching this fork's new default.",
		Migrate:     makeUsersDiscoverable20261020150000,
		Rollback: func(_ *xorm.Engine) error {
			return nil
		},
	})
}
