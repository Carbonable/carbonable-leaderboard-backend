package api

import (
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/carbonable/leaderboard/graph"
	"github.com/carbonable/leaderboard/internal/indexer"
	"github.com/carbonable/leaderboard/internal/starknet"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func GraphqlHandlers(e *echo.Echo, storage indexer.Storage, db *gorm.DB, rpc starknet.StarknetRpcClient) {
	graphqlHandler := handler.NewDefaultServer(
		graph.NewExecutableSchema(
			graph.Config{Resolvers: graph.NewGraphResolver(storage, db, rpc)},
		),
	)
	graphqlHandler.Use(extension.Introspection{})
	playgroundHandler := playground.Handler("GraphQL", "/query")

	e.POST("/query", func(c echo.Context) error {
		graphqlHandler.ServeHTTP(c.Response(), c.Request())
		return nil
	})

	e.GET("/playground", func(c echo.Context) error {
		playgroundHandler.ServeHTTP(c.Response(), c.Request())
		return nil
	})
}
