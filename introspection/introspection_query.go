package introspection

// Copied/extended from https://github.com/graphql/graphiql/blob/master/src/utility/introspectionQueries.js
// Updated for post-June 2018 spec: added specifiedByURL to FullType fragment
// (for @specifiedBy on SCALAR; ensures ComputeSchemaJSON/introspection tests include it).
// Exported as IntrospectionQuery for use in example/main_test.go (specifiedBy verification).
const IntrospectionQuery = `
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
	# specifiedByURL for SCALAR types (e.g., custom DateTime with URL).
	specifiedByURL
	# directives for type-level directives (e.g., @oneOf on INPUT_OBJECT for input
	# unions per spec; also covers scalars/enums if extended). Minimal fields to
	# keep query light (matches __schema.directives); ensures ComputeSchemaJSON
	# and tests (introspection_test.go) include @oneOf etc. (like specifiedByURL
	# and deprecation fields added post-2018).
	directives {
		name
		description
		locations
	}
}
fragment InputValue on __InputValue {
	name
	description
	type { ...TypeRef }
	defaultValue
	# isDeprecated/deprecationReason for input values deprecation support
	# (ARGUMENT_DEFINITION/INPUT_FIELD_DEFINITION spec; matches fields and
	# updated InputValue struct).
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
							ofType {
								kind
								name
							}
						}
					}
				}
			}
		}
	}
}`
