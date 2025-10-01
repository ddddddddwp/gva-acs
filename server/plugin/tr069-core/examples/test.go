package main

import (
"context"
"fmt"
"github.com/root/demo/tr069/factory"
)

func main() {
// Create a parser using factory
p := factory.NewParser()

// Create a builder using factory
b := factory.NewBuilder()

// Test parsing
ctx := context.Background()
_, err := p.ParseMessage(ctx, []byte("<test></test>"))
if err != nil {
fmt.Printf("Parse error: %v\n", err)
} else {
fmt.Println("Parse successful")
}

// Test building
_, err = b.BuildMessage(ctx, nil)
if err != nil {
fmt.Printf("Build error: %v\n", err)
} else {
fmt.Println("Build successful")
}
}
