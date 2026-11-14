package greeter

import "fmt"

// Greet prints a greeting message to the console
func Greet(name string) {
	fmt.Printf("Hello, %s! Welcome to ign.\n", name)
}

// GetGreeting returns a greeting message as a string
func GetGreeting(name string) string {
	return fmt.Sprintf("Hello, %s! Welcome to ign.", name)
}
