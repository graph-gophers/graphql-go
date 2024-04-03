// Package testdata copies validation test cases from the reference implementation at
// github.com/graphql/graphql-js.
//
// Pre-requisites:
// - nodejs
// - npm >= 5.2.0 (for use of npx)
//
// Usage:
// $ go generate .
package testdata

//go:generate npm install
//go:generate npm run export
