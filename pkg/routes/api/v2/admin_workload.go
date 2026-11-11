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

package apiv2

import (
	"context"
	"net/http"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/models"

	"github.com/danielgtaylor/huma/v2"
)

type adminWorkloadBody struct {
	Body *models.Workload
}

// Permissions are enforced by the gateV2AdminRoutes path middleware, not per-handler.
func RegisterAdminWorkloadRoutes(api huma.API) {
	Register(api, huma.Operation{
		OperationID: "admin-workload",
		Summary:     "Workload per assignee (admin)",
		Description: "Groups every open task (not done, not deleted, in a non-archived project) by assignee across all projects. Tasks whose status is Idea are only counted. Hours sum the sizes 1, 4, 8, 20 and 40; unestimated (0) and long-running (999) tasks are counted apart. A task with several assignees appears in each block but counts once in the total. Not paginated. Restricted to instance admins; other callers get a 404.",
		Method:      http.MethodGet,
		Path:        "/admin/workload",
		Tags:        []string{"admin"},
	}, adminWorkload)
}

func init() { AddRouteRegistrar(RegisterAdminWorkloadRoutes) }

func adminWorkload(_ context.Context, _ *struct{}) (*adminWorkloadBody, error) {
	s := db.NewSession()
	defer s.Close()

	w, err := models.GetWorkload(s)
	if err != nil {
		return nil, translateDomainError(err)
	}
	return &adminWorkloadBody{Body: w}, nil
}
