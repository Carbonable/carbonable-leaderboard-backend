package api

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net/http"
	"strconv"

	"github.com/carbonable/leaderboard/internal/indexer"
	"github.com/carbonable/leaderboard/internal/starknet"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func StarknetHandlers(e *echo.Echo, storage indexer.Storage, db *gorm.DB, rpc starknet.StarknetRpcClient) {
	e.GET("/latest-block", func(c echo.Context) error {
		res := storage.Get([]byte("LATEST_BLOCK"))

		buf := bytes.NewBuffer(res)
		decoder := gob.NewDecoder(buf)
		var bn string
		err := decoder.Decode(&bn)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, struct{ error string }{
				error: err.Error(),
			})
		}

		num, _ := strconv.ParseUint(bn, 10, 64)

		return c.JSON(200, struct{ BlockNumber uint64 }{
			BlockNumber: num,
		})
	})

	e.GET("/block/:number", func(c echo.Context) error {
		number, err := strconv.ParseUint(c.Param("number"), 10, 64)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, ApiErrorResponse{
				Error:  err.Error(),
				Reason: "failed to parse block number",
			})
		}

		key := []byte("BLOCK#" + strconv.FormatUint(number, 10))

		if !storage.Has(key) {
			return c.JSON(http.StatusNotFound, ApiErrorResponse{
				Error:  "block not found",
				Reason: "block not found",
			})
		}

		block := storage.Get(key)
		buf := bytes.NewBuffer(block)
		decoder := gob.NewDecoder(buf)
		var resp starknet.GetBlockResponse
		err = decoder.Decode(&resp)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, ApiErrorResponse{
				Error:  fmt.Sprintf("failed to decode block : %s", err.Error()),
				Reason: "failed to decode block",
			})
		}
		return c.JSON(http.StatusOK, struct{ Block starknet.GetBlockResponse }{
			Block: resp,
		})
	})

	e.GET("/contract/:hash", func(c echo.Context) error {
		encodedTxs := storage.Scan([]byte(c.Param("hash") + "#TX#"))
		encodedEvents := storage.Scan([]byte(c.Param("hash") + "#EVENT#"))
		txs, _ := starknet.DecodeSlice[starknet.Transaction](encodedTxs)
		events, _ := starknet.DecodeSlice[starknet.Event](encodedEvents)

		return c.JSON(http.StatusOK, struct {
			Txs    []*starknet.Transaction
			Events []*starknet.Event
		}{
			Txs:    txs,
			Events: events,
		})
	})

	e.GET("/contract-idx/:address", func(c echo.Context) error {
		address := c.Param("address")
		contractIdxKey := []byte(fmt.Sprintf("IDX#%s", address))

		contractIdx := indexer.NewContractIndex(0)
		if storage.Has(contractIdxKey) {
			idx := storage.Get(contractIdxKey)
			err := contractIdx.Decode(idx)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, ApiErrorResponse{
					Error:  "failed to decode contract index",
					Reason: err.Error(),
				})
			}

			return c.JSON(http.StatusOK, contractIdx)
		}

		return c.JSON(http.StatusNotFound, ApiErrorResponse{
			Error:  "contract index not found",
			Reason: "contract index not found",
		})
	})
}
