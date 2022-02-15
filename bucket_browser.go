package common

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"

	"github.com/labstack/echo/v4"
)

type StoreListing struct {
	Stores []string `json:"stores"`
}

type MethodListing struct {
	Methods []string `json:"methods"`
}

type ObjectResponse struct {
	Object  *json.RawMessage `json:"object,omitempty"`
	Objects *json.RawMessage `json:"objects,omitempty"`
}

type BucketBrowserObjectRequest struct {
	Method string `json:"method"`
	Field  string `json:"field"`
}

// BucketBrowser s
type BucketBrowser struct {
	jwtAuthKey     *rsa.PublicKey
	stores         []string
	allowedMethods map[string]reflect.Value
	storageSpec    ServiceStorageSpec
}

func NewBucketBrowser(spec ServiceStorageSpec, jwtKey *rsa.PublicKey, stores ...interface{}) *BucketBrowser {
	bb := &BucketBrowser{
		jwtAuthKey:     jwtKey,
		storageSpec:    spec,
		stores:         make([]string, 0, len(stores)),
		allowedMethods: make(map[string]reflect.Value),
	}

	bb.expose(stores)

	return bb
}

func (bb *BucketBrowser) expose(stores []interface{}) {
	for _, store := range stores {
		t := reflect.ValueOf(store)
		m := t.MethodByName("Backend")
		if !m.IsValid() || m.IsZero() {
			log.Fatalf("cannot reflect Backend method on %T\n", store)
		}

		var storeName string
		out := m.Call([]reflect.Value{})
		if ls, ok := out[0].Interface().(LingioStore); ok {
			storeName = ls.StoreName()
		} else {
			log.Fatalf("cannot get store name from %T\n", store)
		}

		bb.stores = append(bb.stores, storeName)

		var b BucketSpec
		for _, bucket := range bb.storageSpec.Buckets {
			if bucket.BucketName == storeName {
				b = bucket
				break
			}
		}
		if b.BucketName == "" {
			log.Fatalf("cannot find bucket spec for  %s\n", storeName)
		}

		// implicit Get by id
		get := t.MethodByName("Get")
		if !get.IsValid() || get.IsZero() {
			log.Fatalf("cannot find method '%s' on store '%s'\n", "Get", storeName)
		}
		bb.allowedMethods[fqmn(storeName, "Get")] = get

		for _, idx := range b.SecondaryIndexes {
			methodName := IndexMethodName(idx.Type, idx.Name)
			method := t.MethodByName(methodName)
			if !method.IsValid() || method.IsZero() {
				log.Fatalf("cannot find method '%s' on store '%s'\n", methodName, storeName)
			}
			bb.allowedMethods[fqmn(storeName, methodName)] = method
		}
	}
}

func (bb *BucketBrowser) RegisterHandlers(e *echo.Echo) {
	g := e.Group("/ops", bb.allowOnlyAdmins)
	g.GET("/stores", bb.listAllStores)
	g.GET("/stores/:store/methods", bb.listStoreMethods)
	g.POST("/stores/:store/object-requests", bb.getStoreObject)
}

func (bb *BucketBrowser) listAllStores(c echo.Context) error {
	listing := StoreListing{
		Stores: bb.stores,
	}
	return c.JSON(http.StatusOK, listing)
}

func (bb *BucketBrowser) listStoreMethods(c echo.Context) error {
	storeName := c.Param("store")
	var methods []string

	for _, b := range bb.storageSpec.Buckets {
		if b.BucketName == storeName {
			methods = append(methods, "Get")
			for _, idx := range b.SecondaryIndexes {
				methods = append(methods, IndexMethodName(idx.Type, idx.Name))
			}
		}
	}

	return c.JSON(http.StatusOK, MethodListing{
		Methods: methods,
	})
}

func (bb *BucketBrowser) getStoreObject(c echo.Context) error {
	storeName := c.Param("store")

	var req BucketBrowserObjectRequest
	if err := c.Bind(&req); err != nil {
		return fmt.Errorf("binding bucket browser object request: %w", err)
	}

	method, ok := bb.allowedMethods[fqmn(storeName, req.Method)]
	if !ok {
		return RespondError(c, NewError(http.StatusBadRequest).Msg("invalid method"))
	}

	// expects 3 outputs: value, etag, error
	out := method.Call([]reflect.Value{reflect.ValueOf(req.Field)})
	if len(out) != 3 {
		panic("bucket browser: unexpected nbr of outputs")
	}

	if err, ok := out[2].Interface().(*Error); ok && err != nil {
		if err.HttpStatusCode == http.StatusNotFound {
			return RespondError(c, err.Msg("object not found"))
		} else {
			return RespondError(c, err.Msg("internal storage error"))
		}
	}

	data, err := json.Marshal(out[0].Interface())
	if err != nil {
		return RespondError(c, NewErrorE(http.StatusInternalServerError, err))
	}

	if out[0].Kind() == reflect.Array ||
		out[0].Kind() == reflect.Slice {
		c.JSON(http.StatusOK, ObjectResponse{
			Objects: (*json.RawMessage)(&data),
		})
	}

	return c.JSON(http.StatusOK, ObjectResponse{
		Object: (*json.RawMessage)(&data),
	})
}

func (bb *BucketBrowser) allowOnlyAdmins(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		ctx.Set("bearerAuth.Scopes", []string{"admin", "cs"})
		if _, err := AuthCheckCtx(ctx, bb.jwtAuthKey, "", ""); err != nil {
			// Note: technically wrong place to handle error, but it'll have to do for now
			RespondError(ctx, err)
			return nil
		}
		return next(ctx)
	}
}

// fully qualified method name
func fqmn(storeName, methodName string) string {
	return storeName + "." + methodName
}
