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

	"code.vikunja.io/api/pkg/user"
	"code.vikunja.io/api/pkg/web"
	"xorm.io/builder"
	"xorm.io/xorm"
)

// CanUpdate checks if a user can update a label
func (l *Label) CanUpdate(s *xorm.Session, a web.Auth) (bool, error) {
	return canManageLabel(s, a, l.ID)
}

// CanDelete checks if a user can delete a label
func (l *Label) CanDelete(s *xorm.Session, a web.Auth) (bool, error) {
	return canManageLabel(s, a, l.ID)
}

// CanRead checks if a user can read a label
func (l *Label) CanRead(s *xorm.Session, a web.Auth) (bool, int, error) {
	label, err := getLabelByIDSimple(s, l.ID)
	// A plain denial: reads don't disclose whether a label id exists.
	if IsErrLabelDoesNotExist(err) {
		return false, 0, nil
	}
	if err != nil {
		return false, 0, err
	}

	perm, has, err := labelProjectPermission(s, a, label.ProjectID)
	if err != nil || !has {
		return false, 0, err
	}
	return true, int(perm), nil
}

// CanCreate checks if the user can create a label in l.ProjectID.
func (l *Label) CanCreate(s *xorm.Session, a web.Auth) (bool, error) {
	if _, is := a.(*LinkSharing); is {
		return false, nil
	}

	project, err := GetProjectSimpleByID(s, l.ProjectID)
	if err != nil {
		return false, err
	}

	return canWriteLabelProject(s, a, project.ID)
}

// canManageLabel decides on the stored label's project, never on a project_id
// sent in the request body.
func canManageLabel(s *xorm.Session, a web.Auth, labelID int64) (bool, error) {
	if _, is := a.(*LinkSharing); is {
		return false, nil
	}

	label, err := getLabelByIDSimple(s, labelID)
	if err != nil {
		return false, err
	}

	return canWriteLabelProject(s, a, label.ProjectID)
}

func canWriteLabelProject(s *xorm.Session, a web.Auth, projectID int64) (bool, error) {
	perm, has, err := labelProjectPermission(s, a, projectID)
	if err != nil || !has || perm < PermissionWrite {
		return false, err
	}

	// Surfaces ErrProjectIsArchived like Project.CanWrite does.
	if err := (&Project{ID: projectID}).CheckIsArchived(s); err != nil {
		return false, err
	}
	return true, nil
}

// labelProjectPermission is the caller's own membership permission on a
// label's project. It reads the same map as labelVisibleCond, so single-label
// checks and the list cannot drift, and instance admins get no bypass here.
func labelProjectPermission(s *xorm.Session, a web.Auth, projectID int64) (Permission, bool, error) {
	if share, is := a.(*LinkSharing); is {
		return share.Permission, share.ProjectID == projectID, nil
	}

	u, err := user.GetFromAuth(a)
	if err != nil {
		return 0, false, err
	}

	access, err := getProjectAccessForUser(s, u.ID)
	if err != nil {
		return 0, false, err
	}
	perm, has := access.permission(projectID)
	return perm, has, nil
}

// labelVisibleCond matches every label in a project the caller can access. It
// is the one definition of label visibility - single-label reads and the list
// must not drift.
//
// The cond must be handed to Where in one shot. Chaining xorm's session
// .Where/.Or/.And instead flattens the SQL to `A OR B OR C AND D`, which leaked
// any label with any label_tasks row to any authenticated user
// (GHSA-hj5c-mhh2-g7jq).
func labelVisibleCond(s *xorm.Session, a web.Auth) (builder.Cond, error) {

	// Includes projects inherited via a shared parent.
	visible, err := accessibleProjectIDsCond(s, a, "labels.project_id")
	if err != nil {
		return nil, err
	}

	// Both consumers embed this in a builder.And, which silently drops an invalid
	// cond - the query would then match every label instead of failing.
	if visible == nil || !visible.IsValid() {
		return nil, fmt.Errorf("refusing to return an empty label visibility condition for auth %d", a.GetID())
	}

	return visible, nil
}
