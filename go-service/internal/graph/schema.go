package graph

import (
	"github.com/graphql-go/graphql"
	"github.com/lukas/ai-aggregator/go-service/internal/grpcclient"
	"github.com/lukas/ai-aggregator/go-service/internal/store"
)

func NewSchema(s *store.Store, grpcClient *grpcclient.Client) (graphql.Schema, error) {
	queryType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"problems": &graphql.Field{
				Type: graphql.NewNonNull(ProblemConnectionType),
				Args: graphql.FieldConfigArgument{
					"filter":   &graphql.ArgumentConfig{Type: ProblemFilterInput},
					"page":     &graphql.ArgumentConfig{Type: graphql.Int, DefaultValue: 1},
					"pageSize": &graphql.ArgumentConfig{Type: graphql.Int, DefaultValue: 20},
				},
				Resolve: problemsResolver(s),
			},
			"problem": &graphql.Field{
				Type: RawProblemType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
				},
				Resolve: problemResolver(s),
			},
			"clusters": &graphql.Field{
				Type: graphql.NewNonNull(ClusterConnectionType),
				Args: graphql.FieldConfigArgument{
					"page":     &graphql.ArgumentConfig{Type: graphql.Int, DefaultValue: 1},
					"pageSize": &graphql.ArgumentConfig{Type: graphql.Int, DefaultValue: 20},
				},
				Resolve: clustersResolver(s),
			},
			"cluster": &graphql.Field{
				Type: ProblemClusterType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
				},
				Resolve: clusterResolver(s),
			},
			"trends": &graphql.Field{
				Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(TrendSnapshotType))),
				Args: graphql.FieldConfigArgument{
					"windowDays": &graphql.ArgumentConfig{Type: graphql.Int, DefaultValue: 7},
				},
				Resolve: trendsResolver(s),
			},
			"categories": &graphql.Field{
				Type:    graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(CategoryCountType))),
				Resolve: categoriesResolver(s),
			},
			"chatHistory": &graphql.Field{
				Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(ChatMessageType))),
				Args: graphql.FieldConfigArgument{
					"sessionId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
				},
				Resolve: chatHistoryResolver(s),
			},
		},
	})

	mutationType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Mutation",
		Fields: graphql.Fields{
			"createChatSession": &graphql.Field{
				Type:    graphql.NewNonNull(ChatSessionType),
				Resolve: createChatSessionResolver(s),
			},
			"sendMessage": &graphql.Field{
				Type: graphql.NewNonNull(ChatMessageType),
				Args: graphql.FieldConfigArgument{
					"sessionId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
					"content":   &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: sendMessageResolver(s, grpcClient),
			},
		},
	})

	return graphql.NewSchema(graphql.SchemaConfig{
		Query:    queryType,
		Mutation: mutationType,
	})
}
