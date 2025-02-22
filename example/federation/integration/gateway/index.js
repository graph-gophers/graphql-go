import { ApolloServer } from '@apollo/server';
import { startStandaloneServer } from '@apollo/server/standalone';
import { ApolloGateway, IntrospectAndCompose } from '@apollo/gateway';

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
    plugins: [
        ApolloServerPluginSubscription(),
    ],
});

(async () => {
    const { url } = await startStandaloneServer(server);
    console.log(`Server ready at ${url}`);
})();
