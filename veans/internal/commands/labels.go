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

package commands

import (
	"context"
	"strings"

	"code.vikunja.io/veans/internal/client"
)

// labelNamespace is auto-prepended to label names that don't already have it,
// so the agent's labels live in their own corner of the project's label
// list and do not pollute manually-curated labels.
const labelNamespace = "veans:"

func normalizeLabelTitle(raw string) string {
	t := strings.TrimSpace(raw)
	if t == "" {
		return ""
	}
	if strings.HasPrefix(t, labelNamespace) {
		return t
	}
	return labelNamespace + t
}

// getOrCreateLabelByTitle returns the label with the exact title in the given
// project, creating it there if missing. Labels belong to one project, so the
// same title in another project is a different label and is ignored.
func getOrCreateLabelByTitle(ctx context.Context, c *client.Client, projectID int64, title string) (*client.Label, error) {
	existing, err := c.ListLabels(ctx, title)
	if err != nil {
		return nil, err
	}
	for _, l := range existing {
		if l.ProjectID == projectID && l.Title == title {
			return l, nil
		}
	}
	created, err := c.CreateLabel(ctx, &client.Label{Title: title, ProjectID: projectID})
	if err != nil {
		return nil, err
	}
	return created, nil
}

// findLabelOnTask returns the label with the given (already-normalized)
// title attached to the task, or nil. Used by --label-remove to know which
// label ID to detach.
func findLabelOnTask(t *client.Task, title string) *client.Label {
	for _, l := range t.Labels {
		if l != nil && l.Title == title {
			return l
		}
	}
	return nil
}
