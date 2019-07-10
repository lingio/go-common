package log

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/lingio/go-common/logicerr"

	"go.opencensus.io/trace"
)

func Test_create(t *testing.T) {
	ctx := context.Background()
	ll := NewLingioLogger("local", "test", "test")
	partnerID := "project34523452456"
	userID := "user12341234"
	req := &http.Request{RequestURI: "http://some.url"}

	ctx, span := trace.StartSpan(ctx, "Testspan", trace.WithSampler(trace.AlwaysSample()))
	defer span.End()

	vmap := map[string]string{"mapkey": "mapvalue"}

	//errmap := map[string]string{"errorkey": "errorvalue"}
	//err := logicerr.Error{Message: "This is the evil error!", HttpStatusCode: 5000, InfoMap: errmap}
	err := logicerr.NewInternalError("This is the evil error!", nil)

	ll.Debug(ctx, "Woah", nil, nil)
	ll.Info(ctx, "Hello", nil, nil)
	ll.Warning(ctx, "Scary warning", nil, nil)
	ll.Error(ctx, "Error that broke it all", nil, nil)
	fmt.Println()

	ll.Debug(ctx, "Woah", req, nil)
	ll.Info(ctx, "Hello", req, nil)
	ll.Warning(ctx, "Scary warning", req, nil)
	ll.Error(ctx, "Error that broke it all", req, nil)
	fmt.Println()

	ll.Debug(ctx, "Woah", nil, vmap)
	ll.Info(ctx, "Hello", nil, vmap)
	ll.Warning(ctx, "Scary warning", nil, vmap)
	ll.Error(ctx, "Error that broke it all", nil, vmap)
	fmt.Println()

	ll.DebugUser(ctx, "Woah", partnerID, userID, req, nil)
	ll.InfoUser(ctx, "Hello", partnerID, userID, req, nil)
	ll.WarningUser(ctx, "Scary warning", partnerID, userID, req, nil)
	ll.ErrorUser(ctx, "Error that broke it all", partnerID, userID, req, nil)
	fmt.Println()

	ll.DebugUser(ctx, "Woah", partnerID, userID, req, vmap)
	ll.InfoUser(ctx, "Hello", partnerID, userID, req, vmap)
	ll.WarningUser(ctx, "Scary warning", partnerID, userID, req, vmap)
	ll.ErrorUser(ctx, "Error that broke it all", partnerID, userID, req, vmap)
	fmt.Println()

	ll.WarningE(ctx, "Display message for this scary warning!", err, nil)
	ll.ErrorE(ctx, "Display message for this horrible error!", err, nil)
	fmt.Println()

	ll.WarningE(ctx, "Display message for this scary warning!", err, req)
	ll.ErrorE(ctx, "Display message for this horrible error!", err, req)
	fmt.Println()

	ll.WarningUserE(ctx, "Display message for this scary warning!", err, partnerID, userID, req)
	ll.ErrorUserE(ctx, "Display message for this horrible error!", err, partnerID, userID, req)
	fmt.Println()

	ll.Flush()
	ll.Shutdown()
}
