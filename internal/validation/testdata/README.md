# graphql-js testdata

Test cases are generated here by extracting them from [graphql-js] into JSON that we can use to drive Go tests.

## Usage

To update the testdata, run the following command within the `testdata` directory:

```sh
go generate .
```

## How it works

A Node.js project is used to pull in graphql-js as a dependency. The `export.cjs` script runs with Node,
transpiles graphql-js TypeScript files on the fly, loads selected validation test files with a lightweight in-process Mocha shim, and captures expected validation errors into `tests.json`. These test cases in the JSON file are then used to drive the Go tests.

Some upstream validation suites are intentionally disabled in `export.cjs` when
they rely on currently unsupported behavior (for example, subscription-only
fixtures) or known parser/validation parity gaps. Each disabled import has an
inline rationale and should be re-evaluated as parity work progresses.

## Updating dependency

With changes to [graphql-js], update the dependency and regenerate `tests.json`:

```sh
go generate .
```

[graphql-js]: https://github.com/graphql/graphql-js
