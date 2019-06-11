package log

import "testing"

func Test_create(t *testing.T) {
	params := make(map[string]string)
	ll := NewLingioSDL("test", "test", params)
	ll.stdl.Printf("Hello")
}
