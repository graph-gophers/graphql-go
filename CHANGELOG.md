# CHANGELOG

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
