# Implement OneOf Input Objects (Input Unions)

Implement support for OneOf input objects as specified in the September 2025 GraphQL spec.

**Reference:** 
- Parent issue: #688
- Spec PR: https://github.com/graphql/graphql-spec/pull/825

**Description:**
OneOf input objects, also known as 'input unions', allow exactly one field to be specified in an input object. This provides a way to model union-like behavior for input types.

**Implementation Requirements:**
- [ ] Add OneOf directive support in schema parsing
- [ ] Implement validation rules for OneOf input objects
- [ ] Ensure exactly one field is provided in OneOf inputs
- [ ] Add introspection support for OneOf
- [ ] Update documentation and examples

**Related to:** Target Sep 2025 GraphQL spec implementation