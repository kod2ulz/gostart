package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/kod2ulz/gostart/ierrors"
	"github.com/kod2ulz/gostart/errors"
)

// Permission represents a granular permission
type Permission string

// Common permissions
const (
	PermissionRead   Permission = "read"
	PermissionWrite  Permission = "write"
	PermissionDelete Permission = "delete"
	PermissionAdmin  Permission = "admin"
)

// Role represents a user role with permissions
type Role struct {
	ID          uuid.UUID              `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Permissions []Permission           `json:"permissions"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// HasPermission checks if a role has a specific permission
func (r *Role) HasPermission(permission Permission) bool {
	for _, p := range r.Permissions {
		if p == permission || p == PermissionAdmin {
			return true
		}
		// Support wildcard permissions (e.g., "users.*" matches "users.read")
		if strings.HasSuffix(string(p), ".*") {
			prefix := strings.TrimSuffix(string(p), ".*")
			if strings.HasPrefix(string(permission), prefix+".") {
				return true
			}
		}
	}
	return false
}

// AddPermission adds a permission to the role
func (r *Role) AddPermission(permission Permission) {
	if !r.HasPermission(permission) {
		r.Permissions = append(r.Permissions, permission)
	}
}

// RemovePermission removes a permission from the role
func (r *Role) RemovePermission(permission Permission) {
	var filtered []Permission
	for _, p := range r.Permissions {
		if p != permission {
			filtered = append(filtered, p)
		}
	}
	r.Permissions = filtered
}

// UserRole represents a user's role assignment
type UserRole struct {
	UserID uuid.UUID `json:"user_id"`
	RoleID uuid.UUID `json:"role_id"`
}

// RBACStore interface for managing roles and permissions
type RBACStore interface {
	// Role operations
	CreateRole(ctx context.Context, role *Role) error
	GetRole(ctx context.Context, roleID uuid.UUID) (*Role, error)
	GetRoleByName(ctx context.Context, name string) (*Role, error)
	UpdateRole(ctx context.Context, role *Role) error
	DeleteRole(ctx context.Context, roleID uuid.UUID) error
	ListRoles(ctx context.Context) ([]*Role, error)

	// User-Role operations
	AssignRole(ctx context.Context, userID, roleID uuid.UUID) error
	RevokeRole(ctx context.Context, userID, roleID uuid.UUID) error
	GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*Role, error)
	GetRoleUsers(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error)
}

// MemoryRBACStore implements in-memory RBAC storage
type MemoryRBACStore struct {
	roles     map[uuid.UUID]*Role
	userRoles map[uuid.UUID][]uuid.UUID // userID -> roleIDs
	roleUsers map[uuid.UUID][]uuid.UUID // roleID -> userIDs
}

// NewMemoryRBACStore creates a new in-memory RBAC store
func NewMemoryRBACStore() *MemoryRBACStore {
	return &MemoryRBACStore{
		roles:     make(map[uuid.UUID]*Role),
		userRoles: make(map[uuid.UUID][]uuid.UUID),
		roleUsers: make(map[uuid.UUID][]uuid.UUID),
	}
}

// CreateRole creates a new role
func (m *MemoryRBACStore) CreateRole(ctx context.Context, role *Role) error {
	if role.ID == uuid.Nil {
		role.ID = uuid.New()
	}
	m.roles[role.ID] = role
	return nil
}

// GetRole retrieves a role by ID
func (m *MemoryRBACStore) GetRole(ctx context.Context, roleID uuid.UUID) (*Role, error) {
	role, exists := m.roles[roleID]
	if !exists {
		return nil, fmt.Errorf("role not found")
	}
	return role, nil
}

// GetRoleByName retrieves a role by name
func (m *MemoryRBACStore) GetRoleByName(ctx context.Context, name string) (*Role, error) {
	for _, role := range m.roles {
		if role.Name == name {
			return role, nil
		}
	}
	return nil, fmt.Errorf("role not found")
}

// UpdateRole updates an existing role
func (m *MemoryRBACStore) UpdateRole(ctx context.Context, role *Role) error {
	if _, exists := m.roles[role.ID]; !exists {
		return fmt.Errorf("role not found")
	}
	m.roles[role.ID] = role
	return nil
}

// DeleteRole deletes a role
func (m *MemoryRBACStore) DeleteRole(ctx context.Context, roleID uuid.UUID) error {
	delete(m.roles, roleID)
	delete(m.roleUsers, roleID)
	// Remove role from all users
	for userID, roleIDs := range m.userRoles {
		var filtered []uuid.UUID
		for _, id := range roleIDs {
			if id != roleID {
				filtered = append(filtered, id)
			}
		}
		m.userRoles[userID] = filtered
	}
	return nil
}

// ListRoles lists all roles
func (m *MemoryRBACStore) ListRoles(ctx context.Context) ([]*Role, error) {
	roles := make([]*Role, 0, len(m.roles))
	for _, role := range m.roles {
		roles = append(roles, role)
	}
	return roles, nil
}

// AssignRole assigns a role to a user
func (m *MemoryRBACStore) AssignRole(ctx context.Context, userID, roleID uuid.UUID) error {
	// Check if role exists
	if _, exists := m.roles[roleID]; !exists {
		return fmt.Errorf("role not found")
	}

	// Check if already assigned
	for _, id := range m.userRoles[userID] {
		if id == roleID {
			return nil // Already assigned
		}
	}

	m.userRoles[userID] = append(m.userRoles[userID], roleID)
	m.roleUsers[roleID] = append(m.roleUsers[roleID], userID)
	return nil
}

// RevokeRole revokes a role from a user
func (m *MemoryRBACStore) RevokeRole(ctx context.Context, userID, roleID uuid.UUID) error {
	// Remove from user's roles
	var filtered []uuid.UUID
	for _, id := range m.userRoles[userID] {
		if id != roleID {
			filtered = append(filtered, id)
		}
	}
	m.userRoles[userID] = filtered

	// Remove from role's users
	var filteredUsers []uuid.UUID
	for _, id := range m.roleUsers[roleID] {
		if id != userID {
			filteredUsers = append(filteredUsers, id)
		}
	}
	m.roleUsers[roleID] = filteredUsers

	return nil
}

// GetUserRoles retrieves all roles for a user
func (m *MemoryRBACStore) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*Role, error) {
	roleIDs := m.userRoles[userID]
	roles := make([]*Role, 0, len(roleIDs))
	for _, roleID := range roleIDs {
		if role, exists := m.roles[roleID]; exists {
			roles = append(roles, role)
		}
	}
	return roles, nil
}

// GetRoleUsers retrieves all users with a specific role
func (m *MemoryRBACStore) GetRoleUsers(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error) {
	return m.roleUsers[roleID], nil
}

// RBACManager manages role-based access control
type RBACManager struct {
	store RBACStore
}

// NewRBACManager creates a new RBAC manager
func NewRBACManager(store RBACStore) *RBACManager {
	return &RBACManager{store: store}
}

// CreateRole creates a new role
func (r *RBACManager) CreateRole(ctx context.Context, name, description string, permissions []Permission) (*Role, ierrors.Error) {
	role := &Role{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		Permissions: permissions,
	}

	err := r.store.CreateRole(ctx, role)
	if err != nil {
		return nil, errors.Errorf("failed to create role: %w", err)
	}

	return role, nil
}

// GetRole retrieves a role
func (r *RBACManager) GetRole(ctx context.Context, roleID uuid.UUID) (*Role, ierrors.Error) {
	role, err := r.store.GetRole(ctx, roleID)
	if err != nil {
		return nil, errors.Errorf("failed to get role: %w", err)
	}
	return role, nil
}

// AssignRoleToUser assigns a role to a user
func (r *RBACManager) AssignRoleToUser(ctx context.Context, userID, roleID uuid.UUID) ierrors.Error {
	err := r.store.AssignRole(ctx, userID, roleID)
	if err != nil {
		return errors.Errorf("failed to assign role: %w", err)
	}
	return nil
}

// RevokeRoleFromUser revokes a role from a user
func (r *RBACManager) RevokeRoleFromUser(ctx context.Context, userID, roleID uuid.UUID) ierrors.Error {
	err := r.store.RevokeRole(ctx, userID, roleID)
	if err != nil {
		return errors.Errorf("failed to revoke role: %w", err)
	}
	return nil
}

// CheckPermission checks if a user has a specific permission
func (r *RBACManager) CheckPermission(ctx context.Context, userID uuid.UUID, permission Permission) (bool, ierrors.Error) {
	roles, err := r.store.GetUserRoles(ctx, userID)
	if err != nil {
		return false, errors.Errorf("failed to get user roles: %w", err)
	}

	for _, role := range roles {
		if role.HasPermission(permission) {
			return true, nil
		}
	}

	return false, nil
}

// GetUserPermissions returns all permissions for a user
func (r *RBACManager) GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]Permission, ierrors.Error) {
	roles, err := r.store.GetUserRoles(ctx, userID)
	if err != nil {
		return nil, errors.Errorf("failed to get user roles: %w", err)
	}

	permMap := make(map[Permission]bool)
	for _, role := range roles {
		for _, perm := range role.Permissions {
			permMap[perm] = true
		}
	}

	permissions := make([]Permission, 0, len(permMap))
	for perm := range permMap {
		permissions = append(permissions, perm)
	}

	return permissions, nil
}

// RequirePermission middleware for checking permissions
func RequirePermission(rbacManager *RBACManager, permission Permission) func(next func(ctx context.Context) error) func(ctx context.Context) error {
	return func(next func(ctx context.Context) error) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			// Get user from context
			user, ok := ctx.Value(ContextAuthUserKey).(User)
			if !ok {
				return fmt.Errorf("user not found in context")
			}

			// Check permission
			hasPermission, err := rbacManager.CheckPermission(ctx, user.ID(), permission)
			if err != nil {
				return fmt.Errorf("failed to check permission: %w", err)
			}

			if !hasPermission {
				return fmt.Errorf("permission denied: %s", permission)
			}

			return next(ctx)
		}
	}
}

// DefaultRoles provides common role configurations
func DefaultRoles() []*Role {
	return []*Role{
		{
			ID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			Name:        "admin",
			Description: "Administrator with full access",
			Permissions: []Permission{PermissionAdmin},
		},
		{
			ID:          uuid.MustParse("00000000-0000-0000-0000-000000000002"),
			Name:        "user",
			Description: "Standard user with read and write access",
			Permissions: []Permission{PermissionRead, PermissionWrite},
		},
		{
			ID:          uuid.MustParse("00000000-0000-0000-0000-000000000003"),
			Name:        "guest",
			Description: "Guest user with read-only access",
			Permissions: []Permission{PermissionRead},
		},
	}
}

// InitializeDefaultRoles initializes the default roles
func InitializeDefaultRoles(ctx context.Context, store RBACStore) error {
	for _, role := range DefaultRoles() {
		if err := store.CreateRole(ctx, role); err != nil {
			// Ignore errors for existing roles
			continue
		}
	}
	return nil
}
