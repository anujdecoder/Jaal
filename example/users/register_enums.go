package users

import "go.appointy.com/jaal/schemabuilder"

// RegisterEnums registers GraphQL enums (e.g., Role).
// Specific funcs can be added (e.g., RegisterRoleEnum); aggregator follows.
// Pattern from original RegisterEnums in main.go + schemabuilder/types.go.
func RegisterEnums(sb *schemabuilder.Schema) {
	// Role enum registration (ADMIN/MEMBER/GUEST w/ description for spec/Playground).
	// See Role type in users/types.go; sb.Enum(..., desc) per feature (to
	// EnumMapping.Description/__Type.description).
	sb.Enum(RoleMember, map[string]interface{}{
		"ADMIN":  RoleAdmin,
		"MEMBER": RoleMember,
		"GUEST":  RoleGuest,
	}, "Role for user access control (ADMIN full, MEMBER standard, GUEST limited).")
}