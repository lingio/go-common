package log

import (
	"fmt"
	"testing"
)

func Test_create(t *testing.T) {
	ll := NewLingioLogger("local", "test", "test")
	partnerID := "project34523452456"
	userID := "user12341234"

	vmap := map[string]string{"mapkey": "mapvalue"}

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

	ll.Debug1("Woah", "key", "value")
	ll.Info1("Hello", "key", "value")
	ll.Warning1("Scary warning", "key", "value")
	ll.Error1("Error that broke it all", "key", "value")
	fmt.Println()

	ll.Debug2("Woah", "key1", "value1", "key2", "value2")
	ll.Info2("Hello", "key1", "value1", "key2", "value2")
	ll.Warning2("Scary warning", "key1", "value1", "key2", "value2")
	ll.Error2("Error that broke it all", "key1", "value1", "key2", "value2")
	fmt.Println()

	ll.DebugM("Woah", vmap)
	ll.InfoM("Hello", vmap)
	ll.WarningM("Scary warning", vmap)
	ll.ErrorM("Error that broke it all", vmap)
	fmt.Println()

	ll.Flush()
	ll.Shutdown()

}
