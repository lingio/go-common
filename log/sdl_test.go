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

	ll.Debug(ctx, "Woah", nil)
	ll.Info(ctx, "Hello", nil)
	ll.Warning(ctx, "Scary warning", nil)
	ll.Error(ctx, "Error that broke it all", nil)
	fmt.Println()

	ll.Debug(ctx, "Woah", vmap)
	ll.Info(ctx, "Hello", vmap)
	ll.Warning(ctx, "Scary warning", vmap)
	ll.Error(ctx, "Error that broke it all", vmap)
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

	ll.WarningE(ctx, "Display message for this scary warning!", err)
	ll.ErrorE(ctx, "Display message for this horrible error!", err)
	fmt.Println()

	ll.WarningUserE(ctx, "Display message for this scary warning!", err, partnerID, userID, req)
	ll.ErrorUserE(ctx, "Display message for this horrible error!", err, partnerID, userID, req)
	fmt.Println()

	ll.Flush()
	ll.Shutdown()
}
