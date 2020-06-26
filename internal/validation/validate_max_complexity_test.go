package validation

import (
	"testing"

	"github.com/graph-gophers/graphql-go/internal/query"
	"github.com/graph-gophers/graphql-go/internal/schema"
)

type maxComplexityTestCase struct {
	name           string
	query          string
	estimator      ComplexityEstimator
	failure        bool
	expectedErrors []string
}

func (tc maxComplexityTestCase) Run(t *testing.T, s *schema.Schema) {
	t.Run(tc.name, func(t *testing.T) {
		doc, qErr := query.Parse(tc.query)
		if qErr != nil {
			t.Fatal(qErr)
		}

		errs := Validate(s, doc, nil, 0, []ComplexityEstimator{tc.estimator})
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

func TestMaxComplexity(t *testing.T) {
	s := schema.New()

	err := s.Parse(simpleSchema, false)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []maxComplexityTestCase{
		{
			name: "off",
			query: `query Okay { # complexity 0
			characters {         # complexity 1
			  id                 # complexity 2
			  name               # complexity 3
			  friends {          # complexity 4
					friends {    # complexity 5
					  friends {  # complexity 6
						  id     # complexity 7
						  name   # complexity 8
					  }
				  }
			  }
			}
		}`,
			estimator: SimpleEstimator{0},
		},
		{
			name: "maxComplexity-1",
			query: `query Fine {     # complexity 0
				characters {         # complexity 1
				  id                 # complexity 2
				  name               # complexity 3
				  friends {          # complexity 4
					  id             # complexity 5
					  name           # complexity 6
				  }
				}
			}`,
			estimator: SimpleEstimator{7},
		},
		{
			name: "maxComplexity",
			query: `query Equals {   # complexity 0
				characters {         # complexity 1
				  id                 # complexity 2
				  name               # complexity 3
				  friends {          # complexity 4
					  id             # complexity 5
					  name           # complexity 6
				  }
				}
			}`,
			estimator: SimpleEstimator{6},
		},
		{
			name: "maxComplexity+1",
			query: `query Equals {   # complexity 0
				characters {         # complexity 1
				  id                 # complexity 2
				  name               # complexity 3
				  friends {          # complexity 4
					  id             # complexity 5
					  name           # complexity 6
				  }
				}
			}`,
			failure:   true,
			estimator: SimpleEstimator{5},
		},
	} {
		tc.Run(t, s)
	}
}

func TestMaxComplexityRecursion(t *testing.T) {
	s := schema.New()

	err := s.Parse(simpleSchema, false)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []maxComplexityTestCase{
		{
			name: "off",
			query: `query Fine {
				characters {         # complexity 1
				  id
				  name
				  friends {          # complexity 2
					  friends {      # complexity 3
						  friends {  # complexity 4
							  id
							  name
						  }
					  }
				  }
				}
			}`,
			estimator: RecursionEstimator{0},
		},
		{
			name: "maxComplexity",
			query: `query Fine {
				characters {         # complexity 1
				  id
				  name
				  friends {          # complexity 2
					  friends {      # complexity 3
						  friends {  # complexity 4
							  id
							  name
						  }
					  }
				  }
				}
			}`,
			estimator: RecursionEstimator{4},
		},
		{
			name: "maxComplexity + 1",
			query: `query Fine {
				characters {         # complexity 1
				  id
				  name
				  friends {          # complexity 2
					  friends {      # complexity 3
						  friends {  # complexity 4
							  id
							  name
						  }
					  }
				  }
				}
			}`,
			estimator: RecursionEstimator{5},
		},
		{
			name: "number aliases greater then max complexity",
			query: `query Fine {
				characters {         # complexity 1
				  id
				  name
				  friends {          # complexity 2
					  friends {      # complexity 3
						  id
						  name
					  }
				  }
				  favorite: friends {       # complexity 2
					  friends {      		# complexity 3
						  id
						  name
					  }
				  }
				  colleagues: friends {       # complexity 2
					  friends {      		# complexity 3
						  id
						  name
					  }
				  }
				  works: friends {       # complexity 2
					  friends {      	 # complexity 3
						  id
						  name
					  }
				  }
				}
			}`,
			estimator: RecursionEstimator{3},
		},
		{
			name: "maxComplexity - 1",
			query: `query Fine {
				characters {         # complexity 1
				  id
				  name
				  friends {          # complexity 2
					  friends {      # complexity 3
						  friends {  # complexity 4
							  id
							  name
						  }
					  }
				  }
				}
			}`,
			failure:   true,
			estimator: RecursionEstimator{3},
		},
	} {
		tc.Run(t, s)
	}
}

func TestMaxComplexityInlineFragments(t *testing.T) {
	s := schema.New()

	err := s.Parse(interfaceSimple, false)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []maxComplexityTestCase{
		{
			name: "maxComplexity-1",
			query: `query { 		# complexity 0
				characters { 		# complexity 1
				  name 				# complexity 2
				  ... on Human { 	# complexity 3
					totalCredits 	# complexity 4
				  }
				}
			  }`,
			estimator: SimpleEstimator{5},
		},
		{
			name: "maxComplexity",
			query: `query { 		# complexity 0
				characters { 		# complexity 1
				  ... on Droid { 	# complexity 2
					primaryFunction # complexity 3
				  }
				}
			  }`,
			estimator: SimpleEstimator{3},
		},
		{
			name: "maxComplexity+1",
			query: `query { 		# complexity 0
				characters { 		# complexity 1
				  name 				# complexity 2
				  ... on Human { 	# complexity 2
					totalCredits 	# complexity 3
				  }
				}
			  }`,
			failure:   true,
			estimator: SimpleEstimator{2},
		},
	} {
		tc.Run(t, s)
	}
}
func TestMaxComplexityRecursionInlineFragments(t *testing.T) {
	s := schema.New()

	err := s.Parse(interfaceSimple, false)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []maxComplexityTestCase{
		{
			name: "maxComplexity-1",
			query: `query { 		
				characters {		 # depth 1 		
				  name 				
				  ... on Human { 	 
					totalCredits
					friends {		 # depth 2
						name
						friends {    # depth 3
							name 
						}
					}
				  }
				}
			  }`,
			estimator: RecursionEstimator{4},
		},
		{
			name: "maxComplexity",
			query: `query { 		
				characters {		 # depth 1 		
				  name 				
				  ... on Human { 	 
					totalCredits
					friends {		 # depth 2
						name
						friends {    # depth 3
							name 
						}
					}
				  }
				}
			  }`,
			estimator: RecursionEstimator{3},
		},
		{
			name: "maxComplexity + 1",
			query: `query { 		
				characters {		 # depth 1 		
				  name 				
				  ... on Human { 	 
					totalCredits
					friends {		 # depth 2
						name
						friends {    # depth 3
							name 
						}
					}
				  }
				}
			  }`,
			failure:   true,
			estimator: RecursionEstimator{2},
		},
	} {
		tc.Run(t, s)
	}
}

func TestMaxComplexityFragmentSpreads(t *testing.T) {
	s := schema.New()

	err := s.Parse(interfaceSimple, false)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []maxComplexityTestCase{
		{
			name: "maxComplexity-1",
			query: `fragment friend on Character {
				id  								# complexity 7
				name								# complexity 8
				friends {							# complexity 9
					name  							# complexity 10
				}
			}

			query {        				# complexity 0
				characters {         	# complexity 1
				  id                 	# complexity 2
				  name               	# complexity 3
				  friends {          	# complexity 4
					friends {        	# complexity 5
						friends {    	# complexity 6
							...friend 	# complexity 6
						}
					}
				  }
				}
			}`,
			estimator: SimpleEstimator{11},
		},
		{
			name: "maxComplexity",
			query: `fragment friend on Character {
				id 									# complexity 7
				name								# complexity 8
			}
			query {        				# depth 0
				characters {         	# depth 1
				  id                 	# depth 2
				  name               	# depth 3
				  friends {          	# depth 4
					friends {        	# depth 5
						friends {    	# depth 6
							...friend 	# depth 6
						}
					}
				  }
				}
			}`,
			estimator: SimpleEstimator{8},
		},
		{
			name: "maxComplexity+1",
			query: `fragment friend on Character {
				id 									# complexity 8
				name								# complexity 9
				friends {							# complexity 10
					name 							# complexity 11
				}
			}
			query {        					# depth 0
				characters {         		# depth 1
				  id                 		# depth 2
				  name               		# depth 3
				  friends {          		# depth 4
					friends {        		# depth 5
						friends {    		# depth 6
						  friends {  		# depth 7
							...friend 		# depth 7
						  }
						}
					}
				  }
				}
			}`,
			failure:   true,
			estimator: SimpleEstimator{10},
		},
	} {
		tc.Run(t, s)
	}
}

func TestMaxComplexityRecursionFragmentSpreads(t *testing.T) {
	s := schema.New()

	err := s.Parse(interfaceSimple, false)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []maxComplexityTestCase{
		{
			name: "maxComplexity-1",
			query: `fragment friend on Character {
				id  								
				name								
				friends {				# complexity 5			
					name  							
				}
			}

			query {        				# 
				characters {         	# complexity 1
				  id                 	# 
				  name               	# 
				  friends {          	# complexity 2
					friends {        	# complexity 3
						friends {    	# complexity 4
							...friend 	# 
						}
					}
				  }
				}
			}`,
			estimator: RecursionEstimator{6},
		},
		{
			name: "maxComplexity",
			query: `fragment friend on Character {
				id 									
				name								
			}
			query {        				# 
				characters {         	# depth 1
				  id                 	# 
				  name               	# 
				  friends {          	# depth 2
					friends {        	# depth 3
						friends {    	# depth 4
							...friend 	# 
						}
					}
				  }
				}
			}`,
			estimator: RecursionEstimator{4},
		},
		{
			name: "maxComplexity+1",
			query: `fragment friend on Character {
				id 									# 
				name								# 
				friends {							# depth 6
					name 							# 
				}
			}
			query {        					# 
				characters {         		# depth 1
				  id                 		# 
				  name               		# 
				  friends {          		# depth 2
					friends {        		# depth 3
						friends {    		# depth 4
						  friends {  		# depth 5
							...friend 		# 
						  }
						}
					}
				  }
				}
			}`,
			failure: true,
			//expectedErrors: []string{"MaxComplexityExceeded"},
			estimator: RecursionEstimator{5},
		},
	} {
		tc.Run(t, s)
	}
}

func TestMaxComplexityUnknownFragmentSpreads(t *testing.T) {
	s := schema.New()

	err := s.Parse(interfaceSimple, false)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []maxComplexityTestCase{
		{
			name: "maxComplexityUnknownFragment",
			query: `query {        				# complexity 0
				characters {         			# complexity 1
				  id                 			# complexity 2
				  name               			# complexity 3
				  friends {          			# complexity 4
					friends {        			# complexity 5
						friends {    			# complexity 6
						  friends {  			# complexity 7
							...unknownFragment 	# complexity 0
						  }
						}
					}
				  }
				}
			}`,
			estimator:      SimpleEstimator{6},
			failure:        true,
			expectedErrors: []string{"MaxComplexityEvaluationError"},
		},
	} {
		tc.Run(t, s)
	}
}

func TestMaxComplexityValidation(t *testing.T) {
	s := schema.New()

	err := s.Parse(interfaceSimple, false)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name      string
		query     string
		estimator ComplexityEstimator
		expected  bool
	}{
		{
			name: "off",
			query: `query Fine {        # complexity 0
				characters {         	# complexity 1
				  id                 	# complexity 2
				  name               	# complexity 3
				  friends {          	# complexity 4
					  id               	# complexity 5
					  name             	# complexity 6
				  }
				}
			}`,
			estimator: SimpleEstimator{},
		},
		{
			name: "fields",
			query: `query Fine {        # complexity 0
				characters {         	# complexity 1
				  id                 	# complexity 2
				  name               	# complexity 3
				  friends {          	# complexity 4
					  id               	# complexity 5
					  name             	# complexity 6
				  }
				}
			}`,
			expected:  true,
			estimator: SimpleEstimator{5},
		},
		{
			name: "fragment",
			query: `fragment friend on Character {
				id 									# complexity 8
				name								# complexity 9
				friends {							# complexity 10
					name 							# complexity 11
				}
			}
			query {        				# complexity 0
				characters {         	# complexity 1
				  id                 	# complexity 2
				  name               	# complexity 3
				  friends {          	# complexity 4
					friends {        	# complexity 5
						friends {    	# complexity 6
						  friends {  	# complexity 7
							...friend 	# complexity 7
						  }
						}
					}
				  }
				}
			}`,
			expected:  true,
			estimator: SimpleEstimator{10},
		},
		{
			name: "inlinefragment",
			query: `query { 			# complexity 0
				characters { 			# complexity 1
				  ... on Droid { 		# complexity 1
					primaryFunction 	# complexity 2
				  }
				}
			  }`,
			expected:  true,
			estimator: SimpleEstimator{1},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := query.Parse(tc.query)
			if err != nil {
				t.Fatal(err)
			}

			context := newContext(s, doc, 0, []ComplexityEstimator{tc.estimator})
			op := doc.Operations[0]

			opc := &opContext{context: context, ops: doc.Operations}

			actual := validateMaxComplexity(opc, op.Selections)
			if actual != tc.expected {
				t.Errorf("expected %t, actual %t", tc.expected, actual)
			}
		})
	}
}
