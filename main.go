package main

import (
	"fmt"
	"log"
)

func main() {
	log.NewLingioLogger("local", "", "")
	fmt.Println("Hello, world.")
}