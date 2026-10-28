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
	"src.techknowlogick.com/xormigrate"
	"xorm.io/xorm"
)

// New and edited tasks store an unset priority as 0, but older rows can hold NULL. A priority filter such as
// "priority not in 1" or "priority in 0" drops NULL rows, so unset tasks would vanish from "everything but low".
func unsetNullPriorities20261028120000(tx *xorm.Engine) error {
	_, err := tx.Exec("UPDATE tasks SET priority = 0 WHERE priority IS NULL")
	return err
}

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20261028120000",
		Description: "Store unset task priorities as 0 instead of NULL.",
		Migrate:     unsetNullPriorities20261028120000,
		Rollback: func(_ *xorm.Engine) error {
			return nil
		},
	})
}
