package api

import (
	"io"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/handler"
	pb "github.com/lukas/ai-aggregator/go-service/gen/aggregator/v1"
	"github.com/lukas/ai-aggregator/go-service/internal/api/middleware"
	graphDef "github.com/lukas/ai-aggregator/go-service/internal/graph"
	"github.com/lukas/ai-aggregator/go-service/internal/grpcclient"
	"github.com/lukas/ai-aggregator/go-service/internal/store"
)

func NewRouter(s *store.Store, grpcClient *grpcclient.Client) (*gin.Engine, error) {
	schema, err := graphDef.NewSchema(s, grpcClient)
	if err != nil {
		return nil, err
	}

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization", "Cache-Control"},
		AllowCredentials: true,
	}))

	r.Use(middleware.RateLimit(100))

	r.GET("/healthz", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.POST("/graphql", graphqlHandler(schema))
	r.GET("/playground", playgroundHandler())
	r.POST("/chat/stream", chatStreamHandler(s, grpcClient))
	return r, nil
}

func graphqlHandler(schema graphql.Schema) gin.HandlerFunc {
	h := handler.New(&handler.Config{
		Schema:   &schema,
		Pretty:   true,
		GraphiQL: false,
	})
	return func(ctx *gin.Context) {
		h.ServeHTTP(ctx.Writer, ctx.Request)
	}
}

func chatStreamHandler(s *store.Store, grpcClient *grpcclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			SessionID string `json:"sessionId"`
			Content   string `json:"content"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		_, _ = s.CreateChatMessage(c.Request.Context(), req.SessionID, "user", req.Content)

		stream, err := grpcClient.Chat.AskStream(c.Request.Context(), &pb.AskStreamRequest{
			SessionId: req.SessionID,
			Question:  req.Content,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")

		fullAnswer := ""
		c.Stream(func(w io.Writer) bool {
			resp, err := stream.Recv()
			if err != nil {
				return false
			}

			fullAnswer += resp.Content

			if resp.Done {
				_, _ = s.CreateChatMessage(c.Request.Context(), req.SessionID, "assistant", fullAnswer)
				c.SSEvent("done", "")
				return false
			}

			c.SSEvent("message", resp.Content)
			return true
		})
	}
}

func playgroundHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html", []byte(`
  <!DOCTYPE html>
  <html>
  <head>
      <title>GraphQL Playground</title>
      <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/graphql-playground-react/build/static/css/index.css" />
      <script src="https://cdn.jsdelivr.net/npm/graphql-playground-react/build/static/js/middleware.js"></script>
  </head>
  <body>
      <div id="root"></div>
      <script>
          window.addEventListener('load', function () {
              GraphQLPlayground.init(document.getElementById('root'), { endpoint: '/graphql' })
          })
      </script>
  </body>
  </html>`))
	}
}
