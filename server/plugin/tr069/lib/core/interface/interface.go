package _interface

import "context"

type Option func(*Config)

type Config struct {
	StrictMode bool
}

func WithStrictMode(b bool) Option {
	return func(c *Config) {
		c.StrictMode = b
	}
}

type Message struct {
	ID       string
	Method   string
	DeviceID struct {
		SerialNumber string
	}
}

type Parser interface {
	ParseMessage(ctx context.Context, data []byte) (*Message, error)
}

type Builder interface {
	BuildMessage(ctx context.Context, msg *Message) ([]byte, error)
}
