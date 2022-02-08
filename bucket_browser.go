package common

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

type StoreListing struct {
	Stores []string `json:"stores"`
}

type StoreObjectsListing struct {
	Objects []string `json:"objects"`
}

// StoreObject is a json blob composed directly by the getStoreObject method.
// See getStoreObject for more details.
type StoreObject struct {
	Info   ObjectInfo `json:"info"`
	Object []byte     `json:"object"`
}

// BucketBrowser s
type BucketBrowser struct {
	jwtAuthKey *rsa.PublicKey
	backends   []LingioStore
}

func NewBucketBrowser(stores ...LingioStore) *BucketBrowser {
	return &BucketBrowser{
		jwtAuthKey: nil,
		backends:   stores,
	}
}

func (bb *BucketBrowser) WithPublicKey(jwtKey *rsa.PublicKey) *BucketBrowser {
	bb.jwtAuthKey = jwtKey
	return bb
}

func (bb *BucketBrowser) RegisterHandlers(e *echo.Echo) {
	g := e.Group("/ops", bb.allowOnlyAdmins)
	g.GET("/stores", bb.listAllStores)
	g.GET("/stores/:store/objects", bb.listStoreObjects)
	g.GET("/stores/:store/objects/:objectId", bb.getStoreObject)
}

func (bb *BucketBrowser) listAllStores(c echo.Context) error {
	listing := StoreListing{
		Stores: make([]string, 0, len(bb.backends)),
	}

	for _, store := range bb.backends {
		listing.Stores = append(listing.Stores, store.StoreName())

	}
	return c.JSON(http.StatusOK, listing)
}

func (bb *BucketBrowser) listStoreObjects(c echo.Context) error {
	storename := c.Param("store")
	query := c.QueryParam("q")

	for _, store := range bb.backends {
		if store.StoreName() == storename {
			listing := StoreObjectsListing{
				Objects: []string{},
			}

			// Cancel after a reasonable timeout, otherwise some safe amount before write timout
			timeout := 10 * time.Second
			if c.Echo().Server.WriteTimeout > 0 {
				timeout = c.Echo().Server.WriteTimeout - 20*time.Millisecond
			}

			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			for object := range store.ListObjects(ctx) {
				if query == "" || strings.Contains(object.Key, query) {
					listing.Objects = append(listing.Objects, object.Key)
				}
			}

			return c.JSON(http.StatusOK, listing)
		}
	}

	return c.JSON(http.StatusNotFound, nil)
}

func (bb *BucketBrowser) getStoreObject(c echo.Context) error {
	storename := c.Param("store")
	filename := c.Param("objectId")

	for _, store := range bb.backends {
		if store.StoreName() == storename {
			data, info, lerr := store.GetObject(filename)
			if lerr != nil {
				return RespondError(c, lerr)
			}

			infodata, err := json.Marshal(info)
			if err != nil {
				return RespondError(c, NewErrorE(http.StatusInternalServerError, err))
			}

			var response bytes.Buffer

			response.WriteString("{\"object\":")
			response.Write(data)
			response.WriteString(",\"info\":")
			response.Write(infodata)
			response.WriteString("}")

			return c.JSONBlob(http.StatusOK, response.Bytes())
		}
	}

	return c.JSON(http.StatusNotFound, nil)
}

func (bb *BucketBrowser) allowOnlyAdmins(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		ctx.Set("bearerAuth.Scopes", []string{"admin"})
		if _, err := AuthCheckCtx(ctx, bb.jwtAuthKey, "", ""); err != nil {
			// Note: technically wrong place to handle error, but it'll have to do for now
			RespondError(ctx, err)
			return nil
		}
		return next(ctx)
	}
}
