const { ApolloServer }  = require('apollo-server')
const { ApolloGateway, IntrospectAndCompose } = require('@apollo/gateway');

const gateway = new ApolloGateway({
    supergraphSdl: new IntrospectAndCompose({
        subgraphs: [
            { name: 'one', url: 'http://localhost:4001/query' },
            { name: 'two', url: 'http://localhost:4002/query' },
        ],
    }),
});

const server = new ApolloServer({
    gateway,
    subscriptions: false,
});

server.listen().then(({ url }) => {
    console.log(`Server ready at ${url}`);
});
