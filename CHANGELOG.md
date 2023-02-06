CHANGELOG

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
