package data

import (
	"context"
	"slices"
)

// Permissions represent user permission records in the application.
type Permissions []string

// Include checks whether permissions contain code.
func (p *Permissions) Include(code string) bool {
	return slices.Contains(*p, code)
}

// PermissionModelInterface defines the storage operations available for permissions.
type PermissionModelInterface interface {
	// GetAllForUser retrieves all permission codes for a specified user.
	GetAllForUser(ctx context.Context, id int64) (Permissions, error)
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
		return nil, err
	}
	defer rows.Close()

	var permissions Permissions
	for rows.Next() {
		var permission string
		if err := rows.Scan(&permission); err != nil {
			return nil, err
		}

		permissions = append(permissions, permission)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return permissions, nil
}
