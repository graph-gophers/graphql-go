package validation

import (
	"testing"

	"github.com/graph-gophers/graphql-go/internal/query"
	"github.com/graph-gophers/graphql-go/internal/schema"
	"github.com/graph-gophers/graphql-go/types"
)

const (
	simpleSchema = `schema {
		query: Query
	}

	type Query {
		characters: [Character]!
	}

	type Character {
		id: ID!
		name: String!
		friends: [Character]!
	}`
	interfaceSimple = `schema {
		query: Query
	}

	type Query {
		characters: [Character]
	}

	interface Character {
		id: ID!
		name: String!
		friends: [Character]
		appearsIn: [Episode]!
	}

	enum Episode {
		NEWHOPE
		EMPIRE
		JEDI
	}

	type Starship {}

	type Human implements Character {
		id: ID!
		name: String!
		friends: [Character]
		appearsIn: [Episode]!
		starships: [Starship]
		totalCredits: Int
	}

	type Droid implements Character {
		id: ID!
		name: String!
		friends: [Character]
		appearsIn: [Episode]!
		primaryFunction: String
	}`
)

type maxDepthTestCase struct {
	name           string
	query          string
	depth          int
	failure        bool
	expectedErrors []string
}

func (tc maxDepthTestCase) Run(t *testing.T, s *types.Schema) {
	t.Run(tc.name, func(t *testing.T) {
		doc, qErr := query.Parse(tc.query)
		if qErr != nil {
			t.Fatal(qErr)
		}

		errs := Validate(s, doc, nil, tc.depth)
		if len(tc.expectedErrors) > 0 {
			if len(errs) > 0 {
				for _, expected := range tc.expectedErrors {
					found := false
					for _, err := range errs {
						if err.Rule == expected {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected error %v is missing", expected)
					}
				}
			} else {
				t.Errorf("expected errors [%v] are missing", tc.expectedErrors)
			}
		}
		if (len(errs) > 0) != tc.failure {
			t.Errorf("expected failure: %t, actual errors (%d): %v", tc.failure, len(errs), errs)
		}
	})
}

func TestMaxDepth(t *testing.T) {
	s, err := schema.ParseSchema(simpleSchema, false)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []maxDepthTestCase{
		{
			name: "off",
			query: `query Okay {        # depth 0
			characters {         # depth 1
			  id                 # depth 2
			  name               # depth 2
			  friends {          # depth 2
					friends {    # depth 3
					  friends {  # depth 4
						  id       # depth 5
						  name     # depth 5
					  }
				  }
			  }
			}
		}`,
			depth: 0,
		}, {
			name: "maxDepth-1",
			query: `query Fine {        # depth 0
				characters {         # depth 1
				  id                 # depth 2
				  name               # depth 2
				  friends {          # depth 2
					  id               # depth 3
					  name             # depth 3
				  }
				}
			}`,
			depth: 4,
		}, {
			name: "maxDepth",
			query: `query Deep {        # depth 0
				characters {         # depth 1
				  id                 # depth 2
				  name               # depth 2
				  friends {          # depth 2
					  id               # depth 3
					  name             # depth 3
				  }
				}
			}`,
			depth: 3,
		}, {
			name: "maxDepth+1",
			query: `query TooDeep {        # depth 0
				characters {         # depth 1
				  id                 # depth 2
				  name               # depth 2
				  friends {          # depth 2
						friends {    # depth 3
						  friends {  # depth 4
							id       # depth 5
							name     # depth 5
						  }
						}
					}
				}
			}`,
			depth:   4,
			failure: true,
		},
	} {
		tc.Run(t, s)
	}
}

func TestMaxDepthInlineFragments(t *testing.T) {
	s, err := schema.ParseSchema(interfaceSimple, false)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []maxDepthTestCase{
		{
			name: "maxDepth-1",
			query: `query { # depth 0
				characters { # depth 1
				  name # depth 2
				  ... on Human { # depth 2
					totalCredits # depth 2
				  }
				}
			  }`,
			depth: 3,
		},
		{
			name: "maxDepth",
			query: `query { # depth 0
				characters { # depth 1
				  ... on Droid { # depth 2
					primaryFunction # depth 2
				  }
				}
			  }`,
			depth: 2,
		},
		{
			name: "maxDepth+1",
			query: `query { # depth 0
				characters { # depth 1
				  ... on Droid { # depth 2
					primaryFunction # depth 2
				  }
				}
			  }`,
			depth:   1,
			failure: true,
		},
	} {
		tc.Run(t, s)
	}
}

func TestMaxDepthFragmentSpreads(t *testing.T) {
	s, err := schema.ParseSchema(interfaceSimple, false)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []maxDepthTestCase{
		{
			name: "maxDepth-1",
			query: `fragment friend on Character {
				id  # depth 5
				name
				friends {
					name  # depth 6
				}
			}

			query {        # depth 0
				characters {         # depth 1
				  id                 # depth 2
				  name               # depth 2
				  friends {          # depth 2
					friends {        # depth 3
						friends {    # depth 4
							...friend # depth 5
						}
					}
				  }
				}
			}`,
			depth: 7,
		},
		{
			name: "maxDepth",
			query: `fragment friend on Character {
				id # depth 5
				name
			}
			query {        # depth 0
				characters {         # depth 1
				  id                 # depth 2
				  name               # depth 2
				  friends {          # depth 2
					friends {        # depth 3
						friends {    # depth 4
							...friend # depth 5
						}
					}
				  }
				}
			}`,
			depth: 5,
		},
		{
			name: "maxDepth+1",
			query: `fragment friend on Character {
				id # depth 6
				name
				friends {
					name # depth 7
				}
			}
			query {        # depth 0
				characters {         # depth 1
				  id                 # depth 2
				  name               # depth 2
				  friends {          # depth 2
					friends {        # depth 3
						friends {    # depth 4
						  friends {  # depth 5
							...friend # depth 6
						  }
						}
					}
				  }
				}
			}`,
			depth:   6,
			failure: true,
		},
	} {
		tc.Run(t, s)
	}
}

func TestMaxDepthUnknownFragmentSpreads(t *testing.T) {
	s, err := schema.ParseSchema(interfaceSimple, false)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []maxDepthTestCase{
		{
			name: "maxDepthUnknownFragment",
			query: `query {        # depth 0
				characters {         # depth 1
				  id                 # depth 2
				  name               # depth 2
				  friends {          # depth 2
					friends {        # depth 3
						friends {    # depth 4
						  friends {  # depth 5
							...unknownFragment # depth 6
						  }
						}
					}
				  }
				}
			}`,
			depth:          6,
			failure:        true,
			expectedErrors: []string{"MaxDepthEvaluationError"},
		},
	} {
		tc.Run(t, s)
	}
}

func TestMaxDepthValidation(t *testing.T) {
	s, err := schema.ParseSchema(interfaceSimple, false)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name     string
		query    string
		maxDepth int
		expected bool
	}{
		{
			name: "off",
			query: `query Fine {        # depth 0
				characters {         # depth 1
				  id                 # depth 2
				  name               # depth 2
				  friends {          # depth 2
					  id               # depth 3
					  name             # depth 3
				  }
				}
			}`,
			maxDepth: 0,
		}, {
			name: "fields",
			query: `query Fine {        # depth 0
				characters {         # depth 1
				  id                 # depth 2
				  name               # depth 2
				  friends {          # depth 2
					  id               # depth 3
					  name             # depth 3
				  }
				}
			}`,
			maxDepth: 2,
			expected: true,
		}, {
			name: "fragment",
			query: `fragment friend on Character {
				id # depth 6
				name
				friends {
					name # depth 7
				}
			}
			query {        # depth 0
				characters {         # depth 1
				  id                 # depth 2
				  name               # depth 2
				  friends {          # depth 2
					friends {        # depth 3
						friends {    # depth 4
						  friends {  # depth 5
							...friend # depth 6
						  }
						}
					}
				  }
				}
			}`,
			maxDepth: 5,
			expected: true,
		}, {
			name: "inlinefragment",
			query: `query { # depth 0
				characters { # depth 1
				  ... on Droid { # depth 2
					primaryFunction # depth 2
				  }
				}
			  }`,
			maxDepth: 1,
			expected: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := query.Parse(tc.query)
			if err != nil {
				t.Fatal(err)
			}

			context := newContext(s, doc, tc.maxDepth)
			op := doc.Operations[0]

			opc := &opContext{context: context, ops: doc.Operations}

			actual := validateMaxDepth(opc, op.Selections, 1)
			if actual != tc.expected {
				t.Errorf("expected %t, actual %t", tc.expected, actual)
			}
		})
	}
}
