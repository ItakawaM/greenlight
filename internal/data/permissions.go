package data

import (
	"context"
)

// Permission represents a single permission in the application.
type Permission string

// Permissions represent a collection of user permission records in the application.
type Permissions map[Permission]struct{}

// Include checks whether permissions contain code.
func (p Permissions) Include(code Permission) bool {
	_, ok := p[code]
	return ok
}

// PermissionModelInterface defines the storage operations available for permissions.
type PermissionModelInterface interface {
	// GetAllForUser retrieves all permission codes for a specified user.
	GetAllForUser(ctx context.Context, id int64) (Permissions, error)

	// EnsureUserPermissions sets the provided user permissions.
	EnsureUserPermissions(ctx context.Context, id int64, permissionCodes ...Permission) error
}

// PermissionModel implements PermissionModelInterface.
type PermissionModel struct {
	db DBTX
}

// GetAllForUser implements PermissionModelInterface.
func (m *PermissionModel) GetAllForUser(ctx context.Context, id int64) (Permissions, error) {
	statement :=
		`SELECT permissions.code
		FROM permissions
		INNER JOIN users_permissions ON users_permissions.permission_id = permissions.id
		INNER JOIN users ON users_permissions.user_id = users.id
		WHERE users.id = $1;`

	rows, err := m.db.Query(ctx, statement, id)
	if err != nil {
		return nil, handleContextErrors(err)
	}
	defer rows.Close()

	permissions := make(Permissions, 0)
	for rows.Next() {
		var permission string
		if err := rows.Scan(&permission); err != nil {
			return nil, handleContextErrors(err)
		}

		permissions[Permission(permission)] = struct{}{}
	}

	if err := rows.Err(); err != nil {
		return nil, handleContextErrors(err)
	}

	return permissions, nil
}

// EnsureUserPermissions implements PermissionModelInterface.
func (m *PermissionModel) EnsureUserPermissions(ctx context.Context, id int64, permissionCodes ...Permission) error {
	statement :=
		`INSERT INTO users_permissions
		SELECT $1, permissions.id FROM permissions WHERE permissions.code = ANY($2)
		ON CONFLICT DO NOTHING;`

	_, err := m.db.Exec(ctx, statement, id, permissionCodes)
	return handleContextErrors(err)
}
