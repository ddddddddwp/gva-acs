package parser

import (
	"context"
	"fmt"
)

type Message struct {
	ID        string
	Method    string
	DeviceID  struct {
		SerialNumber string
	}
	Events []struct {
		EventCode string
	}
}

type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) ParseMessage(ctx context.Context, data []byte) (*Message, error) {
	// Mock implementation
	fmt.Printf("[SDK] Parsing data: %s\n", string(data))
	return &Message{
		ID:     "12345",
		Method: "Inform",
		DeviceID: struct {
			SerialNumber string
		}{SerialNumber: "MOCK_DEVICE_001"},
		Events: []struct {
			EventCode string
		}{{EventCode: "2 PERIODIC"}},
	}, nil
}
