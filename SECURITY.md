# Security Policy

## Supported Versions

We always try to maintain the library secure and suggest our users to upgrade to the latest stable version. We realize that sometimes this is not possible.

| Version | Supported          |
| ------- | ------------------ |
| 1.x     | :white_check_mark: |
| < 1.0   | :x:                |

## MaxDepth
If you are using the `graphql.MaxDepth` schema option, make sure that you upgrade to version v1.3.0 or higher due to a bug causing security vulnerability in earlier versions.

## Query Limits

Set `graphql.MaxQueryLength` to limit the size of untrusted query documents. Query length limits reduce parser and validation resource usage and protect endpoints from unbounded queries. Apply an HTTP request-body limit as well (e.g. using `http.MaxBytesReader`), because the server receives the entire request before the query length is checked.

## Reporting a Vulnerability

If you find a security vulnerability with this library, please, DO NOT submit a pull request right away. Please, report the issue to @pavelnikolov in the Gophers Slack in a private message.
