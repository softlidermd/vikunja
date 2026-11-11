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

type tasks20261111120100 struct {
	Size int64 `xorm:"bigint not null default 0"`
}

func (tasks20261111120100) TableName() string {
	return "tasks"
}

type savedFilters20261111120100 struct {
	AllProjects bool `xorm:"default false"`
}

func (savedFilters20261111120100) TableName() string {
	return "saved_filters"
}

func addSizeAndAllProjects20261111120100(tx *xorm.Engine) error {
	return partialSync(tx, tasks20261111120100{}, savedFilters20261111120100{})
}

func init() {
	migrations = append(migrations, &xormigrate.Migration{
		ID:          "20261111120100",
		Description: "Add tasks.size and saved_filters.all_projects",
		Migrate:     addSizeAndAllProjects20261111120100,
		Rollback: func(_ *xorm.Engine) error {
			return nil
		},
	})
}
