# graphql-js testdata

Test cases are generated here by extracting them from [graphql-js] into JSON that we can use to drive Go tests.

## Usage

To update the testdata, run the following command within the `testdata` directory:

```sh
go generate .
```

## How it works

A Node.js project is used to pull in graphql-js as a dependency, and automatically patch that via `patch-package`. These
patches replace the `mocha` test functions `describe`, `it`, assertions and the test `harness`. This allows the
expectations to be captured, and written to a JSON file. These test cases in the JSON file are then used to drive the Go
tests.

## Updating patches

With changes to [graphql-js], the patches may need to be updated. To do this, update the `graphql` dependency under
`node_modules`, and sync the patches with the following command:

```sh
npm run create-patches
```

[graphql-js]: https://github.com/graphql/graphql-js
