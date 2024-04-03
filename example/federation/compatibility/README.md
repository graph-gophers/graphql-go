# Apollo Federation Subgraph Compatibility

## Overview
This application was created to demonstrate that the library is fully compatible with [the Apollo Federation Subgraph spec](https://www.apollographql.com/docs/federation/subgraph-spec/).

## Compatibility Results

<table>
<thead>
<tr><th>Federation 1 Support</th><th>Federation 2 Support</th></tr>
</thead>
<tbody>
<tr><td><table><tr><th><code>_service</code></th><td>🟢</td></tr><tr><th><code>@key (single)</code></th><td>🟢</td></tr><tr><th><code>@key (multi)</code></th><td>🟢</td></tr><tr><th><code>@key (composite)</code></th><td>🟢</td></tr><tr><th><code>repeatable @key</code></th><td>🟢</td></tr><tr><th><code>@requires</code></th><td>🟢</td></tr><tr><th><code>@provides</code></th><td>🟢</td></tr><tr><th><code>federated tracing</code></th><td>🔲</td></tr></table></td><td><table><tr><th><code>@link</code></th><td>🟢</td></tr><tr><th><code>@shareable</code></th><td>🟢</td></tr><tr><th><code>@tag</code></th><td>🟢</td></tr><tr><th><code>@override</code></th><td>🟢</td></tr><tr><th><code>@inaccessible</code></th><td>🟢</td></tr><tr><th><code>@composeDirective</code></th><td>🟢</td></tr><tr><th><code>@interfaceObject</code></th><td>🟢</td></tr></table></td></tr>
</tbody>
</table>

><sup>*</sup>This app intentionally does not demonstrate the use of Apollo Tracing since this is not part of the GraphQL spec. However, you can implement it yourself.

## Test it yourself

The application also has the graphiql interface available at `/graphiql` and you can play with the server. Particularly interesting queries are those using the `_entities` resolver and providing different key representations of type `_Any`. Below is a sample query you can play with. In order to run it:
1. Run `go run .`
2. Navigate to http://localhost:4001/graphiql
3. Copy the query below into the GraphiQL UI:
    ```graphql
    query ($representations: [_Any!]!) {
        _entities(representations: $representations) {
            __typename
            ...on DeprecatedProduct { sku package reason }
            ...on Product { id sku createdBy { email name } }
            ...on ProductResearch { study { caseNumber description } }
            ...on User { email name }
        }
    }
    ```
4. Paste this into the variables section:
    ```json
    {
        "representations": [
            {
                "__typename": "DeprecatedProduct",
                "sku": "apollo-federation-v1",
                "package": "@apollo/federation-v1"
            },
            {
                "__typename": "ProductResearch",
                "study": {
                    "caseNumber": "1234"
                }
            },
            { "__typename": "User", "email": "support@apollographql.com" },
            {
                "__typename": "Product",
                "id": "apollo-federation"
            },
            {
                "__typename": "Product",
                "sku": "federation",
                "package": "@apollo/federation"
            },
            {
                "__typename": "Product",
                "sku": "studio",
                "variation": { "id": "platform" }
            }
        ]
    }
    ```
5. After executing the query you should see the following result:
    ```josn
    {
    "data": {
        "_entities": [
        {
            "__typename": "DeprecatedProduct",
            "package": "@apollo/federation-v1",
            "reason": "Migrate to Federation V2"
        },
        {
            "__typename": "ProductResearch",
            "study": {
            "caseNumber": "1234",
            "description": "Federation Study"
            }
        },
        {
            "__typename": "User",
            "email": "support@apollographql.com",
            "name": "Jane Smith"
        },
        {
            "__typename": "Product",
            "id": "apollo-federation",
            "sku": "federation"
        },
        {
            "__typename": "Product",
            "id": "apollo-federation",
            "sku": "federation"
        },
        {
            "__typename": "Product",
            "id": "apollo-studio",
            "sku": "studio"
        }
        ]
    }
    }
    ```
6. In case you want to run the compatiblity tests yourself:
   ```
   npx @apollo/federation-subgraph-compatibility@1.2.1 pm2 --endpoint http://localhost:4001
   ```
