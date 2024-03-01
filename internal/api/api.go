package api

import (
	"net/http"

	"github.com/carbonable/leaderboard/internal/indexer"
	"github.com/carbonable/leaderboard/internal/starknet"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gorm.io/gorm"
)

type ApiErrorResponse struct {
	Error  string
	Reason string
}

func Run(storage indexer.Storage, db *gorm.DB, rpc starknet.StarknetRpcClient) {
	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	e.GET("/ping", func(c echo.Context) error {
		return c.String(http.StatusOK, "Pong !")
	})

	GraphqlHandlers(e, storage, db, rpc)
	StarknetHandlers(e, storage, db, rpc)

	e.Logger.Fatal(e.Start(":8080"))
}
