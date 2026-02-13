# Test Plan for test-example/server.go (Updated for New Schema)

## Overview
This file lists **detailed tests** to cover **all Jaal features** implemented in server.go (new schema with UUID @specifiedBy scalar, Role enum, Node interface, User/DeletedUser, UserResult union, UserStatus, UserIdentifierInput @oneOf, CreateUserInput, Query/Mutation).

Tests use httptest server + GraphQL queries; assert no errors, data structure, values, introspection.

## Detailed Test List (Follow This Order to Implement)
1. **Setup Test Server**:
   - Start httptest.Server with HTTPHandler().
   - Defer close.
   - Helper postQuery(query string) map for JSON POST.

2. **Introspection Tests** (validate schema, new specs):
   - Test full introspection query (from graphql.org introspectionQuery).
     - Assert __schema has Query/Mutation types.
     - Assert directives include @specifiedBy (locations SCALAR, args url) and @oneOf (locations INPUT_OBJECT).
     - Assert types: UUID (SCALAR with specifiedByURL), Role (ENUM), Node (INTERFACE), User/DeletedUser (OBJECT implementing Node), UserResult (UNION), UserIdentifierInput (INPUT_OBJECT with isOneOf: true), etc.
     - Assert ID scalar (built-in), fields on all objects (id, uuid, username, email, role, status, etc.).
   - Test specific __type for:
     - UUID scalar: specifiedByURL present.
     - UserIdentifierInput: isOneOf = true.
     - Node interface: possibleTypes include User/DeletedUser.
     - Unions (UserResult): possibleTypes.

3. **Query Tests** (all queries/fields):
   - me: Assert User with id/uuid/username/email/role/status.
   - user(by: {id: "test"}): Assert UserResult (User or DeletedUser).
   - allUsers: Assert list of User with all fields.
   - Use fragments on Node/UserResult, variables, aliases, directives (@skip/@include) on fields.
   - Test enum args (role: ADMIN).
   - Test list fields (allUsers).
   - Test non-null (id, uuid, username).

4. **Mutation Tests** (fire all, including oneOf):
   - createUser(input: {username: "test", email: "test@example.com"}): Assert User returned.
   - updateUserRole(id, newRole: ADMIN): Assert User.
   - Test oneOf in UserIdentifierInput for user query/mutation.

5. **@oneOf Validation Tests**:
   - Case 1: Success (Exactly one field: id or email in UserIdentifierInput).
   - Case 2: Failure (Two fields provided).
   - Case 3: Failure (Zero fields provided) → error.

6. **@specifiedBy Verification**:
   - Introspection for UUID scalar specifiedByURL.

7. **Edge/Feature Tests**:
   - Custom scalar UUID in fields/args.
   - Interface resolution (fragment on Node).
   - Union resolution (UserResult).
   - Enum serialization (Role in output).
   - Input with oneOf + scalars/enums.
   - Error cases (invalid enum, missing non-null, bad oneOf, non-existent field).
   - Full query with all fields/variables/directives.

8. **Coverage**:
   - All Jaal: scalars/custom/@specifiedBy, queries/mutations, unions/interfaces, oneOf, enums.
   - Assert no panics, correct JSON, spec compliance.
   - Run with -v, check 100% coverage for server.

Implement each in TestFullFeatures subtests, using postQuery + require.

Follow this order in code.