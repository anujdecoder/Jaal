# Test Plan for test-example/server.go

## Overview
This file lists **detailed tests** to cover **all Jaal features** implemented in server.go (Star Wars schema from graphql.org + @specifiedBy on ID, @oneOf on ReviewInput, mutations/queries/interfaces/unions/enums/scalars/inputs).

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
     - Assert types: Character (INTERFACE), Droid/Human (OBJECT implementing Character), Episode (ENUM), ReviewInput (INPUT_OBJECT with isOneOf: true), etc.
     - Assert ID scalar has specifiedByURL.
     - Assert fields on all objects (id, name, friends, appearsIn, primaryFunction, height, etc.).
   - Test specific __type for:
     - ID scalar: specifiedByURL present.
     - ReviewInput: isOneOf = true.
     - Character interface: possibleTypes include Droid/Human.
     - Unions (SearchResult): possibleTypes.

3. **Query Tests** (all queries/fields):
   - hero(episode): Assert returns Character with id/name/appearsIn/... on Droid/Human fragment.
   - character(id): Assert structure.
   - droid(id): Assert Droid fields (primaryFunction).
   - human(id): Assert Human fields (height with arg, mass, starships).
   - starship(id): Assert Starship fields (length with unit arg).
   - reviews(episode): Assert list of Review (stars, commentary).
   - search(text): Assert union SearchResult (Human/Droid/Starship).
   - Use fragments, variables, aliases, directives (@skip/@include) on fields.
   - Test enum args (episode: NEWHOPE).
   - Test list fields (friends, appearsIn, starships).
   - Test non-null (id!).

4. **Mutation Tests** (fire all, including oneOf):
   - createReview(episode, review: {stars: 5}): Assert Review returned (tests @oneOf).
   - rateFilm(episode, rating): Assert Film.
   - updateHumanName(id, name): Assert Human updated.
   - deleteStarship(id): Assert ID returned.
   - Test oneOf invalid: review with both stars/commentary → error.
   - Test oneOf valid edge (only commentary).

5. **Edge/Feature Tests**:
   - Custom scalar ID in args/returns.
   - Interface resolution (fragment on Character).
   - Union resolution (SearchResult).
   - Enum serialization (Episode in output).
   - Input with oneOf + scalars.
   - Error cases (invalid enum, missing non-null, bad oneOf).
   - Pagination (friendsConnection with args).
   - Full query with all fields/variables/directives.

6. **Coverage**:
   - All Jaal: scalars/custom, queries/mutations, unions/interfaces, oneOf/@specifiedBy.
   - Assert no panics, correct JSON, spec compliance.
   - Run with -v, check 100% coverage for server.

Implement each in TestFullFeatures subtests, using postQuery + require.

Follow this order in code.