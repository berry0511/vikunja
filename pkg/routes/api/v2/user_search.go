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
	"strings"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/user"

	"github.com/danielgtaylor/huma/v2"
)

type userListBody struct {
	Body Paginated[*user.User]
}

// RegisterUserSearchRoutes wires the two user-search endpoints onto the Huma API:
// a global search and a per-project search used for share autocomplete.
func RegisterUserSearchRoutes(api huma.API) {
	Register(api, huma.Operation{
		OperationID: "users-search",
		Summary:     "Search users",
		Description: "Searches users by username, name or full email. Matching by name or email requires the target user to have made themselves discoverable, unless both users share an external (OIDC/LDAP) team. Email addresses are never returned.",
		Method:      http.MethodGet,
		Path:        "/users",
		Tags:        []string{"user"},
	}, usersSearch)

	Register(api, huma.Operation{
		OperationID: "projects-users-search",
		Summary:     "Search users with access to a project",
		Description: "Returns the users who can access the project — through ownership, a direct share or a team — optionally filtered by a search string. Intended for share autocomplete. Requires read access to the project.",
		Method:      http.MethodGet,
		Path:        "/projects/{project}/users/search",
		Tags:        []string{"sharing"},
	}, projectUsersSearch)
}

func init() { AddRouteRegistrar(RegisterUserSearchRoutes) }

func usersSearch(ctx context.Context, in *struct {
	Q string `query:"q" doc:"Search query matched against username, name or full email."`
}) (*userListBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	s := db.NewSession()
	defer s.Close()

	currentUser, err := user.GetFromAuth(a)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}

	users, err := user.SearchUsers(s, in.Q, currentUser)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	// SearchUsers retains its discoverability and privacy rules. For partial
	// matches, discover candidate usernames but re-check every candidate through
	// that same policy-aware search instead of returning raw database user rows.
	needle := strings.ToLower(strings.TrimSpace(in.Q))
	if len([]rune(needle)) >= 2 {
		type candidateRow struct { Username string `xorm:"'username'"` }
		candidates := []candidateRow{}
		pattern := "%" + strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(needle) + "%"
		if err := s.SQL("SELECT username FROM users WHERE LOWER(username) LIKE ? ESCAPE '!' OR (discoverable_by_name = ? AND LOWER(name) LIKE ? ESCAPE '!') LIMIT 40", pattern, true, pattern).Find(&candidates); err != nil {
			_ = s.Rollback()
			return nil, translateDomainError(err)
		}
		seen := make(map[int64]bool, len(users))
		for _, match := range users { seen[match.ID] = true }
		for _, candidate := range candidates {
			matches, searchErr := user.SearchUsers(s, candidate.Username, currentUser)
			if searchErr != nil {
				_ = s.Rollback()
				return nil, translateDomainError(searchErr)
			}
			for _, match := range matches {
				if !strings.EqualFold(match.Username, candidate.Username) || seen[match.ID] { continue }
				seen[match.ID] = true
				users = append(users, match)
			}
		}
	}
	if err := s.Commit(); err != nil {
		return nil, translateDomainError(err)
	}

	return &userListBody{Body: NewPaginated(users, int64(len(users)), 1, len(users))}, nil
}

func projectUsersSearch(ctx context.Context, in *struct {
	ProjectID int64  `path:"project"`
	Q         string `query:"q" doc:"Search query matched against username and name."`
}) (*userListBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	s := db.NewSession()
	defer s.Close()

	currentUser, err := user.GetFromAuth(a)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}

	project := &models.Project{ID: in.ProjectID}
	users, canRead, err := models.SearchUsersForProject(s, project, a, currentUser, in.Q)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if !canRead {
		_ = s.Rollback()
		return nil, huma.Error403Forbidden("forbidden")
	}
	if err := s.Commit(); err != nil {
		return nil, translateDomainError(err)
	}

	return &userListBody{Body: NewPaginated(users, int64(len(users)), 1, len(users))}, nil
}
