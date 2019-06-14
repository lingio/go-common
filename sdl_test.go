package log

import (
	"fmt"
	"testing"

	"github.com/lingio/go-common/logicerr"

	"github.com/lingio/go-common/log"
)

func Test_create(t *testing.T) {
	ll := log.NewLingioLogger("local", "test", "test")
	partnerID := "project34523452456"
	userID := "user12341234"

	vmap := map[string]string{"mapkey": "mapvalue"}

	//errmap := map[string]string{"errorkey": "errorvalue"}
	//err := logicerr.Error{Message: "This is the evil error!", HttpStatusCode: 5000, InfoMap: errmap}
	err := logicerr.NewInternalError("This is the evil error!", nil)

	ll.Debug("Woah")
	ll.Info("Hello")
	ll.Warning("Scary warning")
	ll.Error("Error that broke it all")
	fmt.Println()

	ll.DebugUser("Woah", partnerID, userID)
	ll.InfoUser("Hello", partnerID, userID)
	ll.WarningUser("Scary warning", partnerID, userID)
	ll.ErrorUser("Error that broke it all", partnerID, userID)
	fmt.Println()

	ll.DebugM("Woah", vmap)
	ll.InfoM("Hello", vmap)
	ll.WarningM("Scary warning", vmap)
	ll.ErrorM("Error that broke it all", vmap)
	fmt.Println()

	ll.DebugUserM("Woah", partnerID, userID, vmap)
	ll.InfoUserM("Hello", partnerID, userID, vmap)
	ll.WarningUserM("Scary warning", partnerID, userID, vmap)
	ll.ErrorUserM("Error that broke it all", partnerID, userID, vmap)
	fmt.Println()

	ll.ErrorUserE(err, partnerID, userID)
	ll.ErrorE(err)
	fmt.Println()

	ll.Flush()
	ll.Shutdown()

}
