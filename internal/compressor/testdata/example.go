package main

import "fmt"

// This is a comment
type MyStruct struct {
	FieldA int
	FieldB string
}

func (s *MyStruct) MyMethod(val int) string {
	// Method body
	if val > 0 {
		return fmt.Sprintf("Positive: %d", val)
	}
	return "Zero or Negative"
}

func main() {
	// Main function body
	instance := MyStruct{FieldA: 1, FieldB: "test"}
	fmt.Println(instance.MyMethod(5))
	fmt.Println("Hello, world!")
}
