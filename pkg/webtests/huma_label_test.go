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

package webtests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/user"
	"code.vikunja.io/api/pkg/web/handler"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testuser22 is the second bot owner from pkg/db/fixtures/users.yml; user22
// owns bot 24 and is a member of none of the label projects below.
var testuser22 = user.User{ID: 22, Username: "user_bot_owner_b", Issuer: "local"}

// Bot 23 is an admin member of project 44 (users_projects.yml); bot 24 is not.
var testbot23 = user.User{ID: 23, Username: "bot-owner-a-assistant", Issuer: "local", BotOwnerID: 21}
var testbot24 = user.User{ID: 24, Username: "bot-owner-b-assistant", Issuer: "local", BotOwnerID: 22}

// TestHumaLabel mirrors v1's TestProject shape so v2 contract parity is
// readable side-by-side, ported from the model-level matrix in
// pkg/models/label_test.go.
//
// Every label belongs to one project, and the caller's own membership of that
// project decides: read to see it, write to manage it. Fixture topology (see
// pkg/db/fixtures/labels.yml):
//   - project 1 (user1 owns): #1, #2, #4 (created by user2), #5, #7, #13.
//   - project 16 (user1 writes through team 1's share on its parent 33): #10.
//   - projects 21 and 22 (user1 owns, archived): #14, #15, #17.
//   - project 20 (user13 only): #6 (GHSA-hj5c-mhh2-g7jq) and #8 (created by user1).
//   - project 36 (user15): #16. Project 44 (user21, bot 23): #9, #11, #12.
func TestHumaLabel(t *testing.T) {
	testHandler := webHandlerTestV2{
		user:     &testuser1,
		basePath: "/api/v2/labels",
		idParam:  "label",
		t:        t,
	}

	t.Run("ReadAll", func(t *testing.T) {
		t.Run("Normal - exact visible set for user1", func(t *testing.T) {
			rec, err := testHandler.testReadAllWithUser(nil, nil)
			require.NoError(t, err)

			ids := labelIDsFromReadAll(t, rec.Body.Bytes())
			assert.ElementsMatch(t, []int64{1, 2, 4, 5, 7, 10, 13, 14, 15, 17}, ids,
				"ReadAll must return exactly the labels of user1's projects; body: %s", rec.Body.String())
		})

		t.Run("Pagination - full page forces the Count() fallback", func(t *testing.T) {
			rec, err := testHandler.testReadAllWithUser(url.Values{"page": {"1"}, "per_page": {"3"}}, nil)
			require.NoError(t, err)
			ids := labelIDsFromReadAll(t, rec.Body.Bytes())
			assert.Equal(t, []int64{1, 2, 4}, ids,
				"a full page 1 of 3 must slice the first 3 ids in order; body: %s", rec.Body.String())
			assert.EqualValues(t, 10, totalFromReadAll(t, rec.Body.Bytes()),
				"a full unfiltered page must report the true total via Count(); body: %s", rec.Body.String())
		})
		t.Run("Pagination - partial last page uses the start+len shortcut", func(t *testing.T) {
			rec, err := testHandler.testReadAllWithUser(url.Values{"page": {"4"}, "per_page": {"3"}}, nil)
			require.NoError(t, err)
			ids := labelIDsFromReadAll(t, rec.Body.Bytes())
			assert.Equal(t, []int64{17}, ids,
				"the short last page must contain only the trailing id; body: %s", rec.Body.String())
			assert.EqualValues(t, 10, totalFromReadAll(t, rec.Body.Bytes()),
				"a short last page must still report the true total; body: %s", rec.Body.String())
		})
		t.Run("Pagination - page past the end still reports the true total", func(t *testing.T) {
			rec, err := testHandler.testReadAllWithUser(url.Values{"page": {"5"}, "per_page": {"3"}}, nil)
			require.NoError(t, err)
			ids := labelIDsFromReadAll(t, rec.Body.Bytes())
			assert.Empty(t, ids, "a page past the end must return no items; body: %s", rec.Body.String())
			assert.EqualValues(t, 10, totalFromReadAll(t, rec.Body.Bytes()),
				"an empty page with a non-zero offset must fall back to Count(), not report 0 or the offset; body: %s", rec.Body.String())
		})
		t.Run("Pagination - full non-last page respects the q filter", func(t *testing.T) {
			// q keeps 4 labels, so a full page 1 (2 of 4) isn't the last page - start+len would wrongly report 2.
			rec, err := testHandler.testReadAllWithUser(url.Values{"q": {"1,2,4,5"}, "page": {"1"}, "per_page": {"2"}}, nil)
			require.NoError(t, err)
			ids := labelIDsFromReadAll(t, rec.Body.Bytes())
			assert.Equal(t, []int64{1, 2}, ids,
				"a full filtered page 1 of 2 must slice the first 2 filtered ids; body: %s", rec.Body.String())
			assert.EqualValues(t, 4, totalFromReadAll(t, rec.Body.Bytes()),
				"a full non-last page must report the true filtered total, not start+len; body: %s", rec.Body.String())
		})
	})

	t.Run("ReadOne", func(t *testing.T) {
		t.Run("Normal", func(t *testing.T) {
			rec, err := testHandler.testReadOneWithUser(nil, map[string]string{"label": "1"})
			require.NoError(t, err)
			assert.Contains(t, rec.Body.String(), `"title":"Label #1"`)
			assert.Contains(t, rec.Body.String(), `"project_id":1`)
			assert.Contains(t, rec.Body.String(), `"max_permission":`)
			assert.NotEmpty(t, rec.Result().Header.Get("ETag"))
		})
		t.Run("Nonexisting", func(t *testing.T) {
			// Missing labels return 403, not 404 — the CanRead branch refuses to disclose existence.
			_, err := testHandler.testReadOneWithUser(nil, map[string]string{"label": "9999"})
			require.Error(t, err)
			assert.Equal(t, http.StatusForbidden, getHTTPErrorCode(err))
		})
		t.Run("Permissions check", func(t *testing.T) {
			t.Run("Forbidden - GHSA private label in a foreign project (#6)", func(t *testing.T) {
				_, err := testHandler.testReadOneWithUser(nil, map[string]string{"label": "6"})
				require.Error(t, err)
				assert.Equal(t, http.StatusForbidden, getHTTPErrorCode(err))
			})
			t.Run("Forbidden - own label in a foreign project (#8)", func(t *testing.T) {
				// Creating a label grants nothing; only the project counts.
				_, err := testHandler.testReadOneWithUser(nil, map[string]string{"label": "8"})
				require.Error(t, err)
				assert.Equal(t, http.StatusForbidden, getHTTPErrorCode(err))
			})
			t.Run("Allowed - another user's label in a member project (#4)", func(t *testing.T) {
				rec, err := testHandler.testReadOneWithUser(nil, map[string]string{"label": "4"})
				require.NoError(t, err)
				assert.Contains(t, rec.Body.String(), `"title":"Label #4 - visible via other task"`)
			})
			t.Run("Allowed - project reached through a parent share (#10)", func(t *testing.T) {
				rec, err := testHandler.testReadOneWithUser(nil, map[string]string{"label": "10"})
				require.NoError(t, err)
				assert.Contains(t, rec.Body.String(), `"title":"Label #10 - attached in child project only"`)
			})
			t.Run("Allowed - only on a soft-deleted task (#13)", func(t *testing.T) {
				rec, err := testHandler.testReadOneWithUser(nil, map[string]string{"label": "13"})
				require.NoError(t, err)
				assert.Contains(t, rec.Body.String(), `"title":"Label #13 - attached only to a soft-deleted task"`)
			})
		})
	})

	t.Run("Create", func(t *testing.T) {
		t.Run("Normal", func(t *testing.T) {
			rec, err := testHandler.testCreateWithUser(nil, nil, `{"title":"Lorem","description":"Ipsum","hex_color":"00ff00","project_id":1}`)
			require.NoError(t, err)
			assert.Equal(t, http.StatusCreated, rec.Code)
			assert.Contains(t, rec.Body.String(), `"title":"Lorem"`)
			assert.Contains(t, rec.Body.String(), `"description":"Ipsum"`)
			assert.Contains(t, rec.Body.String(), `"hex_color":"00ff00"`)
			assert.Contains(t, rec.Body.String(), `"project_id":1`)
		})
		t.Run("Hex color is normalized", func(t *testing.T) {
			// NormalizeHex strips a leading '#' (and truncates to 6 chars).
			// Send a non-normalized value and assert the stored/returned form.
			rec, err := testHandler.testCreateWithUser(nil, nil, `{"title":"Normalized","hex_color":"#aabbcc","project_id":1}`)
			require.NoError(t, err)
			assert.Equal(t, http.StatusCreated, rec.Code)
			assert.Contains(t, rec.Body.String(), `"hex_color":"aabbcc"`,
				"leading '#' must be normalized away; body: %s", rec.Body.String())
			assert.NotContains(t, rec.Body.String(), `#aabbcc`)
		})
		t.Run("Empty title", func(t *testing.T) {
			// v2 returns 422, not v1's 400; full body shape asserted in TestHuma_ErrorShapeIsRFC9457.
			_, err := testHandler.testCreateWithUser(nil, nil, `{"title":"","project_id":1}`)
			require.Error(t, err)
			assert.Equal(t, http.StatusUnprocessableEntity, getHTTPErrorCode(err))
		})
		t.Run("Without a project", func(t *testing.T) {
			_, err := testHandler.testCreateWithUser(nil, nil, `{"title":"Lorem"}`)
			require.Error(t, err)
			assert.Equal(t, http.StatusNotFound, getHTTPErrorCode(err))
		})
		t.Run("Missing project", func(t *testing.T) {
			_, err := testHandler.testCreateWithUser(nil, nil, `{"title":"Lorem","project_id":99999}`)
			require.Error(t, err)
			assert.Equal(t, http.StatusNotFound, getHTTPErrorCode(err))
		})
		t.Run("Read-only project", func(t *testing.T) {
			_, err := testHandler.testCreateWithUser(nil, nil, `{"title":"Lorem","project_id":3}`)
			require.Error(t, err)
			assert.Equal(t, http.StatusForbidden, getHTTPErrorCode(err))
		})
		t.Run("Foreign project", func(t *testing.T) {
			_, err := testHandler.testCreateWithUser(nil, nil, `{"title":"Lorem","project_id":20}`)
			require.Error(t, err)
			assert.Equal(t, http.StatusForbidden, getHTTPErrorCode(err))
		})
	})

	t.Run("Update", func(t *testing.T) {
		t.Run("Normal", func(t *testing.T) {
			rec, err := testHandler.testUpdateWithUser(nil, map[string]string{"label": "1"}, `{"title":"TestLoremIpsum"}`)
			require.NoError(t, err)
			assert.Contains(t, rec.Body.String(), `"title":"TestLoremIpsum"`)
		})
		t.Run("project_id in the body is ignored", func(t *testing.T) {
			rec, err := testHandler.testUpdateWithUser(nil, map[string]string{"label": "1"}, `{"title":"Moved?","project_id":16}`)
			require.NoError(t, err)
			assert.Contains(t, rec.Body.String(), `"project_id":1,`)
		})
		t.Run("Nonexisting", func(t *testing.T) {
			_, err := testHandler.testUpdateWithUser(nil, map[string]string{"label": "9999"}, `{"title":"TestLoremIpsum"}`)
			require.Error(t, err)
			assert.Equal(t, http.StatusNotFound, getHTTPErrorCode(err))
		})
		t.Run("Permissions check", func(t *testing.T) {
			t.Run("Allowed - another user's label in a member project (#4)", func(t *testing.T) {
				rec, err := testHandler.testUpdateWithUser(nil, map[string]string{"label": "4"}, `{"title":"TestLoremIpsum"}`)
				require.NoError(t, err)
				assert.Contains(t, rec.Body.String(), `"title":"TestLoremIpsum"`)
			})
			t.Run("Forbidden - GHSA private label (#6)", func(t *testing.T) {
				_, err := testHandler.testUpdateWithUser(nil, map[string]string{"label": "6"}, `{"title":"TestLoremIpsum"}`)
				require.Error(t, err)
				assert.Equal(t, http.StatusForbidden, getHTTPErrorCode(err))
			})
			t.Run("Forbidden - archived project (#14)", func(t *testing.T) {
				_, err := testHandler.testUpdateWithUser(nil, map[string]string{"label": "14"}, `{"title":"TestLoremIpsum"}`)
				require.Error(t, err)
				assert.Equal(t, http.StatusPreconditionFailed, getHTTPErrorCode(err))
			})
		})
	})

	t.Run("Delete", func(t *testing.T) {
		t.Run("Normal", func(t *testing.T) {
			rec, err := testHandler.testDeleteWithUser(nil, map[string]string{"label": "2"})
			require.NoError(t, err)
			// v2 delete is 204 No Content; v1 returned 200 + a message body.
			assert.Equal(t, http.StatusNoContent, rec.Code)
			assert.Empty(t, rec.Body.String())
		})
		t.Run("Nonexisting", func(t *testing.T) {
			_, err := testHandler.testDeleteWithUser(nil, map[string]string{"label": "9999"})
			require.Error(t, err)
			assert.Equal(t, http.StatusNotFound, getHTTPErrorCode(err))
		})
		t.Run("Permissions check", func(t *testing.T) {
			t.Run("Allowed - another user's label in a member project (#4)", func(t *testing.T) {
				rec, err := testHandler.testDeleteWithUser(nil, map[string]string{"label": "4"})
				require.NoError(t, err)
				assert.Equal(t, http.StatusNoContent, rec.Code)
			})
			t.Run("Forbidden - GHSA private label (#6)", func(t *testing.T) {
				_, err := testHandler.testDeleteWithUser(nil, map[string]string{"label": "6"})
				require.Error(t, err)
				assert.Equal(t, http.StatusForbidden, getHTTPErrorCode(err))
			})
		})
	})
}

func TestHumaLabel_ReadOnlyMember(t *testing.T) {
	// Team 1 reads project 21; user2 created label #17 there but may not manage it.
	reader := webHandlerTestV2{
		user:     &testuser2,
		basePath: "/api/v2/labels",
		idParam:  "label",
		t:        t,
	}

	rec, err := reader.testReadOneWithUser(nil, map[string]string{"label": "17"})
	require.NoError(t, err)
	assert.Contains(t, rec.Body.String(), `"title":"Label #5"`)

	_, err = reader.testUpdateWithUser(nil, map[string]string{"label": "17"}, `{"title":"renamed"}`)
	require.Error(t, err)
	assert.Equal(t, http.StatusForbidden, getHTTPErrorCode(err))

	_, err = reader.testDeleteWithUser(nil, map[string]string{"label": "17"})
	require.Error(t, err)
	assert.Equal(t, http.StatusForbidden, getHTTPErrorCode(err))
}

// v1 is frozen but still serves PUT /labels, which now needs a project too.
func TestLabel_V1CreateRequiresProject(t *testing.T) {
	testHandler := webHandlerTest{
		user: &testuser1,
		strFunc: func() handler.CObject {
			return &models.Label{}
		},
		t: t,
	}

	_, err := testHandler.testCreateWithUser(nil, nil, `{"title":"no project"}`)
	require.Error(t, err)
	assert.Equal(t, http.StatusNotFound, getHTTPErrorCode(err))

	rec, err := testHandler.testCreateWithUser(nil, nil, `{"title":"with project","project_id":1}`)
	require.NoError(t, err)
	assert.Contains(t, rec.Body.String(), `"project_id":1`)
}

// Bot labels live in project 44 (owned by user21, bot 23 is an admin member);
// access follows that membership like for any other user.
func TestHumaLabel_BotProjectMembership(t *testing.T) {
	owner := webHandlerTestV2{
		user:     &testuser21,
		basePath: "/api/v2/labels",
		idParam:  "label",
		t:        t,
	}
	require.NoError(t, owner.ensureEnv())
	bot := webHandlerTestV2{
		user:     &testbot23,
		basePath: "/api/v2/labels",
		idParam:  "label",
		t:        t,
		e:        owner.e,
	}
	otherOwner := webHandlerTestV2{
		user:     &testuser22,
		basePath: "/api/v2/labels",
		idParam:  "label",
		t:        t,
		e:        owner.e,
	}
	otherBot := webHandlerTestV2{
		user:     &testbot24,
		basePath: "/api/v2/labels",
		idParam:  "label",
		t:        t,
		e:        owner.e,
	}

	t.Run("ReadAll - members list the project's labels", func(t *testing.T) {
		for _, h := range []webHandlerTestV2{owner, bot} {
			rec, err := h.testReadAllWithUser(nil, nil)
			require.NoError(t, err)
			assert.ElementsMatch(t, []int64{9, 11, 12}, labelIDsFromReadAll(t, rec.Body.Bytes()),
				"user %d; body: %s", h.user.ID, rec.Body.String())
		}
	})
	t.Run("ReadAll - non-members see none of them", func(t *testing.T) {
		for _, h := range []webHandlerTestV2{otherOwner, otherBot} {
			rec, err := h.testReadAllWithUser(nil, nil)
			require.NoError(t, err)
			assert.Empty(t, labelIDsFromReadAll(t, rec.Body.Bytes()), "user %d; body: %s", h.user.ID, rec.Body.String())
		}
	})
	t.Run("ReadOne - a non-member owner of another bot cannot read", func(t *testing.T) {
		_, err := otherOwner.testReadOneWithUser(nil, map[string]string{"label": "9"})
		require.Error(t, err)
		assert.Equal(t, http.StatusForbidden, getHTTPErrorCode(err))
	})
	t.Run("Update - the project owner can rename a bot's label", func(t *testing.T) {
		rec, err := owner.testUpdateWithUser(nil, map[string]string{"label": "9"}, `{"title":"renamed by owner"}`)
		require.NoError(t, err)
		assert.Contains(t, rec.Body.String(), `"title":"renamed by owner"`)
	})
	t.Run("Update - the bot member can rename its owner's label", func(t *testing.T) {
		rec, err := bot.testUpdateWithUser(nil, map[string]string{"label": "11"}, `{"title":"renamed by bot"}`)
		require.NoError(t, err)
		assert.Contains(t, rec.Body.String(), `"title":"renamed by bot"`)
	})
	t.Run("Update - a non-member cannot", func(t *testing.T) {
		_, err := otherOwner.testUpdateWithUser(nil, map[string]string{"label": "9"}, `{"title":"hijack"}`)
		require.Error(t, err)
		assert.Equal(t, http.StatusForbidden, getHTTPErrorCode(err))
	})
	t.Run("Delete - a non-member cannot", func(t *testing.T) {
		_, err := otherOwner.testDeleteWithUser(nil, map[string]string{"label": "9"})
		require.Error(t, err)
		assert.Equal(t, http.StatusForbidden, getHTTPErrorCode(err))
	})

	attach := webHandlerTestV2{
		user:     &testbot23,
		basePath: "/api/v2/tasks/52/labels",
		idParam:  "label",
		t:        t,
		e:        owner.e,
	}
	t.Run("Create - the bot can attach labels of the task's project", func(t *testing.T) {
		for _, labelID := range []string{"11", "12"} {
			rec, err := attach.testCreateWithUser(nil, nil, `{"label_id":`+labelID+`}`)
			require.NoError(t, err)
			assert.Equal(t, http.StatusCreated, rec.Code)
			assert.Contains(t, rec.Body.String(), `"label_id":`+labelID)
		}
	})
	t.Run("Create - the bot cannot attach a label of another project", func(t *testing.T) {
		_, err := attach.testCreateWithUser(nil, nil, `{"label_id":6}`)
		require.Error(t, err)
		assert.Equal(t, http.StatusForbidden, getHTTPErrorCode(err))
	})
}

// labelIDsFromReadAll extracts the label IDs from a v2 paginated list body so
// the visible set can be asserted exactly rather than via substring matching.
func labelIDsFromReadAll(t *testing.T, body []byte) []int64 {
	t.Helper()
	var resp struct {
		Items []struct {
			ID int64 `json:"id"`
		} `json:"items"`
	}
	require.NoError(t, json.Unmarshal(body, &resp), "ReadAll body must be a paginated envelope: %s", string(body))
	ids := make([]int64, 0, len(resp.Items))
	for _, it := range resp.Items {
		ids = append(ids, it.ID)
	}
	return ids
}

func totalFromReadAll(t *testing.T, body []byte) int64 {
	t.Helper()
	var resp struct {
		Total int64 `json:"total"`
	}
	require.NoError(t, json.Unmarshal(body, &resp), "ReadAll body must be a paginated envelope: %s", string(body))
	return resp.Total
}

// The two tests below cover v2-only behaviour with no v1 counterpart:
// ETag + conditional requests, and AutoPatch (merge-patch+json).

func TestHumaLabel_ETagReturns304(t *testing.T) {
	e, err := setupTestEnv()
	require.NoError(t, err)
	token := humaTokenFor(t, &testuser1)

	rec := humaRequest(t, e, http.MethodGet, "/api/v2/labels/1", "", token, "")
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
	etag := rec.Header().Get("ETag")
	require.NotEmpty(t, etag, "GET must return an ETag header")

	req := httptest.NewRequest(http.MethodGet, "/api/v2/labels/1", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("If-None-Match", etag)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusNotModified, rec.Code, "body: %s", rec.Body.String())
}

func TestHumaLabel_ETagReflectsPermission(t *testing.T) {
	// Label #10 is in project 16: user6 owns it (admin), user1 writes through a
	// parent share. Same label, so the per-caller ETag must differ — else a 304 serves stale perms.
	e, err := setupTestEnv()
	require.NoError(t, err)

	reader := humaRequest(t, e, http.MethodGet, "/api/v2/labels/10", "", humaTokenFor(t, &testuser1), "")
	require.Equal(t, http.StatusOK, reader.Code, "body: %s", reader.Body.String())
	owner := humaRequest(t, e, http.MethodGet, "/api/v2/labels/10", "", humaTokenFor(t, &testuser6), "")
	require.Equal(t, http.StatusOK, owner.Code, "body: %s", owner.Body.String())

	assert.NotEmpty(t, reader.Header().Get("ETag"))
	assert.NotEqual(t, reader.Header().Get("ETag"), owner.Header().Get("ETag"),
		"same label, different caller permission must produce different ETags")
}

func TestHumaLabel_PATCHMergePatch(t *testing.T) {
	e, err := setupTestEnv()
	require.NoError(t, err)
	token := humaTokenFor(t, &testuser1)

	// Create a fresh label so we don't stomp fixtures.
	rec := humaRequest(t, e, http.MethodPost, "/api/v2/labels",
		`{"title":"before","description":"keep me","hex_color":"112233","project_id":1}`, token, "")
	require.Equal(t, http.StatusCreated, rec.Code, "body: %s", rec.Body.String())
	var created struct {
		ID int64 `json:"id"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))

	// PATCH only title; AutoPatch must leave description + hex_color alone.
	// Reuses the same echo.Echo so the create above isn't wiped by a fixture reload.
	rec = humaRequest(t, e, http.MethodPatch, fmt.Sprintf("/api/v2/labels/%d", created.ID),
		`{"title":"after"}`, token, "application/merge-patch+json")
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	rec = humaRequest(t, e, http.MethodGet, fmt.Sprintf("/api/v2/labels/%d", created.ID), "", token, "")
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
	var after struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		HexColor    string `json:"hex_color"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &after))
	assert.Equal(t, "after", after.Title)
	assert.Equal(t, "keep me", after.Description, "description must survive the PATCH")
	assert.Equal(t, "112233", after.HexColor, "hex_color must survive the PATCH")
}
