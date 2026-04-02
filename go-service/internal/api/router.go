package api

import (
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/handler"
	"github.com/lukas/ai-aggregator/go-service/internal/api/middleware"
	graphDef "github.com/lukas/ai-aggregator/go-service/internal/graph"
	"github.com/lukas/ai-aggregator/go-service/internal/store"
)

func NewRouter(s *store.Store) (*gin.Engine, error) {
	schema, err := graphDef.NewSchema(s)
	if err != nil {
		return nil, err
	}

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	r.Use(middleware.RateLimit(100))

	r.GET("/healthz", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.POST("/graphql", graphqlHandler(schema))
	r.GET("/playground", playgroundHandler())
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
