# Pets Example

This example is meant to show a full implementation of the server using an SQL datastore. 

### Starting

start the server

```
go run *.go
```

and then visit the GraphiQL dev server at `localhost:8080`

## NOTES:

- int32 maps to the graphql type Int, so if a number is desired, int32 must be used

- the data structs are used for both database storage and graphql resolution.

- resolver methods are all caps because the gorm ORM needs camel cased exported struct
  fields in order to create database columns correctly. Another differentiation scheme can
  be used if desired

- authentication could go in middleware in the server part

- authorization could be done with middleware, or from the user in the context after
  authentication in either the db methods or the resolvers. Official graphql recommends
  that these go into the data layer, but if a per-field authorization is needed, the field
  resolvers are the perfect place.

- edges, connections, used for pagination
    - there is a type
    - it has a connection, which is another type that it is connected to
    - the connection happens through an edge which represents the relationship


### Example queries

get user by ids with pets belonging to the user...

```
{
  getUser(id: 1) {
    name
    pets {
      name
      id
    }
  }
}
```

get pet by id and owner...

```
{
 getPet(id: 1) {
  name
  owner {
    name
    id
  }
}
}
```

insert pet with owner

```
mutation addPet($userID: Int!, $pet: PetInput!) {
  addPet(userID: $userID, pet: $pet) {
    name
  }
}

# query variables...
{
  "userID": 1,
  "pet": {
    "name": "slkdjsaldkjsalkj"
  }
}
```

update pet (needs query variables)
```
mutation UpdatePet($pet: PetInput!) {
  updatePet(pet: $pet) {
    name
    id
    owner {
      name
    }
    tags {
      title
    }
  }

}
```

delete pet (needs query variables)
```
mutation DeletePet($userID: Int!, $petID: Int!) {
  deletePet(userID: $userID, petID: $petID) {
    
  }
}

query Pet {
  one: getPet(id: 1) {
    name
    owner {
      name
    }
    tags {
      title
    }
  },
  two: 	getUser(id: 1) {
    name
  }
}
```

pagination ("after" or "before" can be added with the encoded cursor)
```
{
  getUser(id: 1) {
    name
    petsConnection(first: 2) {
      totalCount
      edges {
        cursor 
        node {
          name
        }
      }
    }
  }
}
```
