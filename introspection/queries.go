package introspection

// SpecVersion represents the GraphQL spec version to use for introspection
type SpecVersion string

const (
	// Spec2018 is the June 2018 GraphQL spec
	Spec2018 SpecVersion = "2018"
	// Spec2021 is the October 2021 GraphQL spec
	Spec2021 SpecVersion = "2021"
	// Spec2025 is the September 2025 GraphQL spec (default)
	Spec2025 SpecVersion = "2025"
)

// GetIntrospectionQuery returns the appropriate introspection query for the given spec version
func GetIntrospectionQuery(version SpecVersion) string {
	switch version {
	case Spec2018:
		return IntrospectionQuery2018
	case Spec2021:
		return IntrospectionQuery2021
	case Spec2025:
		return IntrospectionQuery
	default:
		return IntrospectionQuery
	}
}

// IntrospectionQuery2018 is the introspection query for June 2018 spec
// It excludes fields that were added in later specs:
// - specifiedByURL (added in 2021)
// - isDeprecated/deprecationReason on InputValue (added in 2021)
// - directives on __Type (added in 2021)
const IntrospectionQuery2018 = `
query IntrospectionQuery {
	__schema {
		queryType { name }
		mutationType { name }
		subscriptionType { name }
		types {
			...FullType
		}
		directives {
			name
			description
			locations
			args {
				...InputValue
			}
		}
	}
}
fragment FullType on __Type {
	kind
	name
	description
	fields(includeDeprecated: true) {
		name
		description
		args {
			...InputValue
		}
		type {
			...TypeRef
		}
		isDeprecated
		deprecationReason
	}
	inputFields {
		...InputValue
	}
	interfaces {
		...TypeRef
	}
	enumValues(includeDeprecated: true) {
		name
		description
		isDeprecated
		deprecationReason
	}
	possibleTypes {
		...TypeRef
	}
}
fragment InputValue on __InputValue {
	name
	description
	type { ...TypeRef }
	defaultValue
}
fragment TypeRef on __Type {
	kind
	name
	ofType {
		kind
		name
		ofType {
			kind
			name
			ofType {
				kind
				name
				ofType {
					kind
					name
					ofType {
						kind
						name
						ofType {
							kind
							name
						}
					}
				}
			}
		}
	}
}`

// IntrospectionQuery2021 is the introspection query for October 2021 spec
// It adds:
// - specifiedByURL (added in 2021)
// - isDeprecated/deprecationReason on InputValue (added in 2021)
// But excludes:
// - directives on __Type (added in 2025)
const IntrospectionQuery2021 = `
query IntrospectionQuery {
	__schema {
		queryType { name }
		mutationType { name }
		subscriptionType { name }
		types {
			...FullType
		}
		directives {
			name
			description
			locations
			args {
				...InputValue
			}
		}
	}
}
fragment FullType on __Type {
	kind
	name
	description
	fields(includeDeprecated: true) {
		name
		description
		args {
			...InputValue
		}
		type {
			...TypeRef
		}
		isDeprecated
		deprecationReason
	}
	inputFields {
		...InputValue
	}
	interfaces {
		...TypeRef
	}
	enumValues(includeDeprecated: true) {
		name
		description
		isDeprecated
		deprecationReason
	}
	possibleTypes {
		...TypeRef
	}
	specifiedByURL
}
fragment InputValue on __InputValue {
	name
	description
	type { ...TypeRef }
	defaultValue
	isDeprecated
	deprecationReason
}
fragment TypeRef on __Type {
	kind
	name
	ofType {
		kind
		name
		ofType {
			kind
			name
			ofType {
				kind
				name
				ofType {
					kind
					name
					ofType {
						kind
						name
						ofType {
							kind
							name
						}
					}
				}
			}
		}
	}
}`

// ParseSpecVersion parses a string into a SpecVersion
func ParseSpecVersion(s string) SpecVersion {
	switch s {
	case "2018":
		return Spec2018
	case "2021":
		return Spec2021
	case "2025":
		return Spec2025
	default:
		return Spec2025
	}
}
