package main

import (
	"fmt"

	"github.com/tacogips/ign/internal/greeter"
)

func main() {
	// Call the greeter package function
	greeter.Greet("World")

	// Get greeting message as a string
	message := greeter.GetGreeting("Go Developer")
	fmt.Println(message)
}
