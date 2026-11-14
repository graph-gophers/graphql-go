# CHANGELOG

## Unreleased

* [IMPROVEMENT] Reduce the memory needed to return large scalars. Values of 64KiB or more are no longer copied into every enclosing field's buffer, and large strings are escaped once, directly into the response. Returning a 16MiB string nested three fields deep now allocates about the size of the response instead of 7-8x that. Output stays byte-identical to `json.Marshal`.
* [BUGFIX] Reject string descriptions before schema and type extensions instead of silently discarding them.
* [BUGFIX] Reject malformed or out-of-range numeric literals during parsing instead of allowing panics. Custom scalars continue to receive numeric literals within the supported int64 and float64 ranges.

[v1.11.0](https://github.com/graph-gophers/graphql-go/releases/tag/v1.11.0) Release v1.11.0


* [FEATURE] Add the `MaxSelections(n)` schema option to change the limit on expanded selections per operation. Zero and negative values select the default (100,000); use a very large value such as `math.MaxInt` for effectively no limit.
* [BUGFIX] Check max depth for every depth at which a fragment is spread, not only the first.
* [BUGFIX] Pass the `TraceField` context to every resolver.
* [BUGFIX] Stop the subscription resolver timeout from sending after the subscription context is cancelled.

[v1.10.3](https://github.com/graph-gophers/graphql-go/releases/tag/v1.10.3) Release v1.10.3

* [BUGFIX] Fail validation for very long queries.
* [BUGFIX] Limit the lexer recursion depth.
* [BUGFIX] Enforce `@oneOf` for values supplied through variables.
* [BUGFIX] Reject undefined input object fields supplied via variables.
* [BUGFIX] Preserve the Go interface type when resolving fields (#763).
* [BUGFIX] Fix `TraceField` finish function never receiving the resolver's error.

[v1.10.2](https://github.com/graph-gophers/graphql-go/releases/tag/v1.10.2) Release v1.10.2

* [FEATURE] Add `ValidateDeprecated()` schema option to enable validation of deprecated fields, arguments (including directive arguments), input fields, and enum values in queries. When enabled, usage of deprecated schema elements results in validation errors. Schemas created with `Clone` inherit the setting.

[v1.10.1](https://github.com/graph-gophers/graphql-go/releases/tag/v1.10.1) Release v1.10.1

* [BUGFIX] Fix panics on recursive interfaces.
* [BUGFIX] Fix resolver cycles where one of the types is a union.

[v1.10.0](https://github.com/graph-gophers/graphql-go/releases/tag/v1.10.0) Release v1.10.0

* [FEATURE] Add the `@oneOf` input object directive.
* [FEATURE] Allow `int64` custom scalars.
* [FEATURE] Support executable-document description strings on full-form operations, fragments, and variable definitions. Descriptions remain non-semantic and do not change validation or execution behavior. Executable `#` comments remain ignored.
* [FEATURE] Support full Unicode in the lexer.
* [FEATURE] Validate that deprecated interface fields are consistent in implementations.
* [IMPROVEMENT] Improve validation performance and parity with graphql-js.
* [IMPROVEMENT] Enforce `MaxQueryLength` for subscriptions.
* [BUGFIX] Fix introspection: deprecation reasons, `isDeprecated`, `includeDeprecated` on arguments and input fields, and default values.
* [BUGFIX] Reject empty queries at parse time.
* [BUGFIX] Disallow `@skip` and `@include` on top-level subscription fields.
* [BUGFIX] Fix the trace context passed to tracers.

[v1.9.0](https://github.com/graph-gophers/graphql-go/releases/tag/v1.9.0) Release v1.9.0

* [IMPROVEMENT] Reduce query execution allocations by reusing internal temporary buffers in the execution hot path. Add `DisableMemoryPooling()` schema option to opt out and enable pooled vs non-pooled benchmark comparison. Added a `MaxPooledBufferCap(n)` method to set the maximum buffer capacity (in bytes) that can be returned to the internal memory pool. The default limit is 16KB.
* [FEATURE] Allow schema cloning and applying a resolver to a schema without one. See `Clone`, `MustClone` and `ApplyResolver` schema methods for more details.
* [CHORE] Applied `go fix ./...`-style modernization across the repo to align the code with newer Go idioms and standard library helpers.
* [CHORE] Bump Go version in go.mod file to v1.25 to be one minor version less than the latest stable Go release.

[v1.8.0](https://github.com/graph-gophers/graphql-go/releases/tag/v1.8.0) Release v1.8.0

* [FEATURE] Added `DecodeSelectedFieldArgs` helper function to decode argument values for any (nested) selected field path directly from a resolver context, enabling efficient multi-level prefetching without per-resolver argument reflection. This enables selective, multi‑level batching (Category → Products → Reviews) by loading only requested fields, mitigating N+1 issues despite complex filters or pagination.
* [CHORE] Bump Go version in go.mod file to v1.24 to be one minor version less than the latest stable Go release.

[v1.7.2](https://github.com/graph-gophers/graphql-go/releases/tag/v1.7.2) Release v1.7.2

* [BUGFIX] Fix checksum mismatch between direct git access and golang proxy for v1.7.1. This version contains identical functionality to v1.7.1 but with proper tag creation to ensure consistent checksums across all proxy configurations.

[v1.7.1](https://github.com/graph-gophers/graphql-go/releases/tag/v1.7.1) Release v1.7.1

* [IMPROVEMENT] `SelectedFieldNames` now returns dot-delimited nested field paths (e.g. `products`, `products.id`, `products.category`, `products.category.id`). Intermediate container object/list paths are included so resolvers can check for both a branch (`products.category`) and its leaves (`products.category.id`). `HasSelectedField` and `SortedSelectedFieldNames` operate on these paths. This aligns behavior with typical resolver projection needs and fixes missing nested selections.
* [BUGFIX] Reject object, interface, and input object type definitions that declare zero fields/input values (spec compliance).
* [IMPROVEMENT] Optimize overlapping field validation to avoid quadratic memory blowups on large sibling field lists.
* [FEATURE] Add configurable safety valve for overlapping field comparison count with `OverlapValidationLimit(n)` schema option (0 disables the cap). When exceeded validation aborts early with rule `OverlapValidationLimitExceeded`. Disabled by default.
* [TEST] Add benchmarks & randomized overlap stress test for mixed field/fragment patterns.

[v1.7.0](https://github.com/graph-gophers/graphql-go/releases/tag/v1.7.0) Release v1.7.0

* [FEATURE] Add resolver field selection inspection helpers (`SelectedFieldNames`, `HasSelectedField`, `SortedSelectedFieldNames`). Helpers are available by default and compute results lazily only when called. An explicit opt-out (`DisableFieldSelections()` schema option) is provided for applications that want to remove even the minimal context insertion overhead when the helpers are never used.

[v1.5.0](https://github.com/graph-gophers/graphql-go/releases/tag/v1.5.0) Release v1.5.0

* [FEATURE] Add specifiedBy directive in #532
* [IMPROVEMENT] In this release we improve validation for primitive values, directives, repeat directives, #515, #516, #525, #527
* [IMPROVEMENT] Fix minor unreachable code caused by t.Fatalf #530
* [BUG] Fix __type queries sometimes not returning data in #540
* [BUG] Allow deprecated directive on arguments by @pavelnikolov in #541
* [DOCS] Add array input example #536

[v1.4.0](https://github.com/graph-gophers/graphql-go/releases/tag/v1.4.0) Release v1.4.0

* [FEATURE] Add basic first step for Apollo Federation. This does NOT include full subgraph specification. This PR adds support only for `_service` schema level field. This library is long way from supporting the full sub-graph spec and we do not plan to implement that any time soon.

[v1.3.0](https://github.com/graph-gophers/graphql-go/releases/tag/v1.3.0) Release v1.3.0

* [FEATURE] Support custom panic handler #468
* [FEATURE] Support interfaces implementing interfaces #471
* [BUG] Support parsing nanoseconds time properly #486
* [BUG] Fix a bug in maxDepth fragment spread logic #492

[v1.2.0](https://github.com/graph-gophers/graphql-go/releases/tag/v1.2.0) Release v1.2.0

* [DOCS] Added examples of how to add JSON map as input scalar type. The goal of this change was to improve documentation #467

[v1.1.0](https://github.com/graph-gophers/graphql-go/releases/tag/v1.1.0) Release v1.1.0

* [FEATURE] Add types package #437
* [FEATURE] Expose `packer.Unmarshaler` as `decode.Unmarshaler` to the public #450
* [FEATURE] Add location fields to type definitions #454
* [FEATURE] `errors.Errorf` preserves original error similar to `fmt.Errorf` #456
* [BUGFIX] Fix duplicated __typename in response (fixes #369) #443

[v1.0.0](https://github.com/graph-gophers/graphql-go/releases/tag/v1.0.0) Initial release
