package main

import (
	"context"
	"fmt"
	"log"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/factory"
)

func main() {
	// Create parser and builder
	parser := factory.NewParser()
	builder := factory.NewBuilder()

	// Register a custom RPC method
	err := parser.RegisterCustomMethod("CustomMethod", func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		// Process the custom method with parameters
		result := map[string]interface{}{
			"status":    "success",
			"processed": len(params),
			"message":   "Custom method executed successfully",
		}
		return result, nil
	})
	if err != nil {
		log.Fatalf("Failed to register custom method: %v", err)
	}

	// Check if the method is registered
	if parser.IsCustomMethodRegistered("CustomMethod") {
		fmt.Println("CustomMethod is registered")
	}

	// List all registered custom methods
	methods := parser.ListCustomMethods()
	fmt.Printf("Registered custom methods: %v\n", methods)

	// Create a sample result for demonstration
	sampleResult := map[string]interface{}{
		"result": "Custom response data",
		"status": "completed",
	}

	// Build a response for the custom method
	ctx := context.Background()
	response, err := builder.BuildCustomRPCResponse(ctx, "CustomMethod", sampleResult)
	if err != nil {
		log.Fatalf("Failed to build custom RPC response: %v", err)
	}

	fmt.Printf("Custom RPC Response: %s\n", string(response))

	// Unregister the custom method
	err = parser.UnregisterCustomMethod("CustomMethod")
	if err != nil {
		log.Fatalf("Failed to unregister custom method: %v", err)
	}

	fmt.Println("CustomMethod unregistered")
}