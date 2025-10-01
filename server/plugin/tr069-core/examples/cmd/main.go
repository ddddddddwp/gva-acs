package main

import (
        "fmt"
        "github.com/root/demo/tr069/examples/examples"
)

func main() {
        fmt.Println("TR069 Library Examples")
        fmt.Println("======================")

        // Run basic example
        fmt.Println("\n1. Running Basic Example:")
        examples.Example()

        // Run security example
        fmt.Println("\n2. Running Security Example:")
        examples.SecurityExample()

        // Run full security example
        fmt.Println("\n3. Running Full Security Example:")
        examples.FullSecurityExample()
}