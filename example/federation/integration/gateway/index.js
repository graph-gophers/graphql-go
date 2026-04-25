import { ApolloServer } from '@apollo/server';
import { startStandaloneServer } from '@apollo/server/standalone';
import { ApolloServerPluginLandingPageLocalDefault } from '@apollo/server/plugin/landingPage/default';
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
        ApolloServerPluginLandingPageLocalDefault({
            embed: true,
            defaultDocument: `query ExampleQuery {
  hi
  hello
}
`,
        }),
    ],
});

(async () => {
    const { url } = await startStandaloneServer(server);
    console.log(`Server ready at ${url}`);
})();
