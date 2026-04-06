package users

import "go.appointy.com/jaal/schemabuilder"

// RegisterEnums registers GraphQL enums (e.g., Role).
// Specific funcs can be added (e.g., RegisterRoleEnum); aggregator follows.
// Pattern from original RegisterEnums in main.go + schemabuilder/types.go.
func RegisterEnums(sb *schemabuilder.Schema) {
	// Role enum registration (ADMIN/MEMBER/GUEST w/ description for spec/Playground).
	// See Role type in users/types.go; sb.Enum(..., option) per feature (to
	// EnumMapping.Description/__Type.description).
	// Also demonstrates enum value deprecation: GUEST is deprecated, use MEMBER instead.
	sb.Enum(RoleMember, map[string]interface{}{
		"ADMIN":  RoleAdmin,
		"MEMBER": RoleMember,
		"GUEST":  RoleGuest,
	}, schemabuilder.WithDescription("Role for user access control (ADMIN full, MEMBER standard, GUEST limited)."),
		schemabuilder.EnumValueDeprecation("GUEST", "Use MEMBER instead. Guest access is being phased out."))

	// Status enum with deprecated value example
	sb.Enum(StatusActive, map[string]interface{}{
		"ACTIVE":    StatusActive,
		"INACTIVE":  StatusInactive,
		"PENDING":   StatusPending,
		"SUSPENDED": StatusSuspended,
	}, schemabuilder.WithDescription("User account status."),
		schemabuilder.EnumValueDeprecation("SUSPENDED", "Use INACTIVE instead. Suspended status will be removed in v2.0."))
}
