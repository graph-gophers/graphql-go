// Package testdata copies validation test cases from the Javascript reference implementation.
//
// Usage:
// $ git clone github.com/graphql/graphql-js
// $ go generate internal/validation/testdata
package testdata

//go:generate cp export.js graphql-js/export.js
//go:generate babel-node graphql-js/export.js
