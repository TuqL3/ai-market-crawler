package graph

import (
	"context"

	"github.com/graphql-go/graphql"
	"github.com/lukas/ai-aggregator/go-service/internal/store"
)

func problemsResolver(s *store.Store) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		page, _ := p.Args["page"].(int)
		pageSize, _ := p.Args["pageSize"].(int)

		filter := make(map[string]interface{})
		if f, ok := p.Args["filter"].(map[string]interface{}); ok {
			filter = f
		}

		problems, total, err := s.GetProblems(context.Background(), filter, page, pageSize)
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"items":      problems,
			"totalCount": total,
			"page":       page,
			"pageSize":   pageSize,
		}, nil
	}
}

func problemResolver(s *store.Store) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		id, _ := p.Args["id"].(string)
		return s.GetProblemByID(context.Background(), id)
	}
}
func clustersResolver(s *store.Store) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		page, _ := p.Args["page"].(int)
		pageSize, _ := p.Args["pageSize"].(int)

		clusters, total, err := s.GetClusters(context.Background(), page, pageSize)
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"items":      clusters,
			"totalCount": total,
			"page":       page,
			"pageSize":   pageSize,
		}, nil
	}
}

func clusterResolver(s *store.Store) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		id, _ := p.Args["id"].(string)
		return s.GetClusterByID(context.Background(), id)
	}
}

func trendsResolver(s *store.Store) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		windowDays, _ := p.Args["windowDays"].(int)
		return s.GetTrends(context.Background(), windowDays)
	}
}

func categoriesResolver(s *store.Store) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		results, err := s.GetCategories(context.Background())
		if err != nil {
			return nil, err
		}
		var categories []map[string]interface{}
		for _, r := range results {
			categories = append(categories, map[string]interface{}{
				"category": r["category"],
				"count":    r["count"],
			})
		}
		return categories, nil
	}
}

func chatHistoryResolver(s *store.Store) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		sessionID, _ := p.Args["sessionId"].(string)
		return s.GetChatHistory(context.Background(), sessionID)
	}
}
func createChatSessionResolver(s *store.Store) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		return s.CreateChatSession(context.Background())
	}
}

func sendMessageResolver(s *store.Store) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		sessionID, _ := p.Args["sessionId"].(string)
		content, _ := p.Args["content"].(string)

		// Lưu message của user
		_, err := s.CreateChatMessage(context.Background(), sessionID, "user", content)
		if err != nil {
			return nil, err
		}

		// TODO: Phase 5 sẽ gọi gRPC ChatService.Ask ở đây
		// Tạm thời trả về placeholder response
		return s.CreateChatMessage(context.Background(), sessionID, "assistant", "Chat will be implemented in Phase 5")
	}
}
