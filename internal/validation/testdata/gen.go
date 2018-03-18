// Package testdata copies validation test cases from the reference implementation at
// github.com/graphql/graphql-js.
//
// Pre-requisites:
// - nodejs
// - npm >= 5.2.0 (for use of npx)
//
// Usage:
// $ git clone https://github.com/graphql/graphql-js
// $ go generate .
package testdata

//go:generate npm install
//go:generate cp export.js graphql-js/export.js
//go:generate npx babel-node graphql-js/export.js
