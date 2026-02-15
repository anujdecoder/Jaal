package users

import "go.appointy.com/jaal/schemabuilder"

// RegisterEnums registers GraphQL enums (e.g., Role).
// Specific funcs can be added (e.g., RegisterRoleEnum); aggregator follows.
// Pattern from original RegisterEnums in main.go + schemabuilder/types.go.
func RegisterEnums(sb *schemabuilder.Schema) {
	// Role enum registration (ADMIN/MEMBER/GUEST).
	// See Role type in main.go; sb.Enum maps strings to values for introspection.
	sb.Enum(RoleMember, map[string]interface{}{
		"ADMIN":  RoleAdmin,
		"MEMBER": RoleMember,
		"GUEST":  RoleGuest,
	})
}