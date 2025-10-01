// Package stream implements the TR069 stream parser for handling large XML messages.
// 包 stream 实现了 TR069 流式解析器，用于处理大型 XML 消息。
package stream

import (
	"bufio"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"runtime"
	"strings"
	"sync/atomic"

	"github.com/root/demo/tr069/interfaces"
)

// streamParser implements the StreamParser interface.
// streamParser 实现了 StreamParser 接口。
type streamParser struct {
	bufferSize    int
	maxMemory     int64
	currentMemory int64
	strictMode    bool
}

// NewStreamParser creates a new stream parser with default settings.
// NewStreamParser 创建一个新的流式解析器，默认设置。
func NewStreamParser() interfaces.StreamParser {
	return &streamParser{
		bufferSize: 4096,            // Default buffer size
		// 默认缓冲区大小
		maxMemory: 10 * 1024 * 1024, // Default max memory: 10MB
		// 默认最大内存：10MB
		strictMode: false,           // Default strict mode: off
		// 默认严格模式：关闭
	}
}

// ParseStream parses a TR069 message from a stream.
// ParseStream 从流中解析 TR069 消息。
func (s *streamParser) ParseStream(ctx context.Context, reader io.Reader) (*interfaces.Message, error) {
	// Reset memory usage counter
	s.ResetMemoryUsage()
	
	// Create a buffered reader
	bufReader := bufio.NewReaderSize(reader, s.bufferSize)
	
	// Create a decoder
	decoder := xml.NewDecoder(bufReader)
	
	// Create a message
	message := &interfaces.Message{}
	
	// Parse the stream
	for {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		
		// Check memory limit
		if s.maxMemory > 0 && atomic.LoadInt64(&s.currentMemory) > s.maxMemory {
			return nil, errors.New("memory limit exceeded")
		}
		
		// Parse the next token
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error parsing XML: %w", err)
		}
		
		// Update memory usage
		s.updateMemoryUsage(token)
		
		// Process the token
		switch t := token.(type) {
		case xml.StartElement:
			// Handle start element
			if t.Name.Local == "Envelope" {
				message.EnvelopeStart = true
			} else if t.Name.Local == "Header" {
				message.HasHeader = true
			} else if t.Name.Local == "Body" {
				message.HasBody = true
			}
			
			// Process attributes
			for _, attr := range t.Attr {
				if s.strictMode && !isValidAttribute(attr.Name.Local, attr.Value) {
					return nil, fmt.Errorf("invalid attribute: %s", attr.Name.Local)
				}
				
				// Add attribute to message
				if t.Name.Local == "Envelope" {
					message.EnvelopeAttrs = append(message.EnvelopeAttrs, interfaces.Attribute{
						Name:  attr.Name.Local,
						Value: attr.Value,
					})
				}
			}
			
		case xml.EndElement:
			// Handle end element
			if t.Name.Local == "Envelope" {
				message.EnvelopeEnd = true
			}
			
		case xml.CharData:
			// Handle character data
			data := string(t)
			if len(strings.TrimSpace(data)) > 0 {
				// Add content to message
				message.Content = append(message.Content, data)
			}
		}
	}
	
	// Validate message
	if s.strictMode {
		if !message.EnvelopeStart || !message.EnvelopeEnd {
			return nil, errors.New("invalid SOAP envelope")
		}
		if !message.HasBody {
			return nil, errors.New("missing SOAP body")
		}
	}
	
	return message, nil
}

// ParseStreamWithCallback parses a TR069 message from a stream and calls the callback
// for each parsed element.
func (s *streamParser) ParseStreamWithCallback(ctx context.Context, reader io.Reader, callback interfaces.ParseCallback) error {
	// Reset memory usage counter
	s.ResetMemoryUsage()
	
	// Create a buffered reader
	bufReader := bufio.NewReaderSize(reader, s.bufferSize)
	
	// Create a decoder
	decoder := xml.NewDecoder(bufReader)
	
	// Parse the stream
	for {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		// Check memory limit
		if s.maxMemory > 0 && atomic.LoadInt64(&s.currentMemory) > s.maxMemory {
			return errors.New("memory limit exceeded")
		}
		
		// Parse the next token
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error parsing XML: %w", err)
		}
		
		// Update memory usage
		s.updateMemoryUsage(token)
		
		// Process the token and call callback
		switch t := token.(type) {
		case xml.StartElement:
			element := &interfaces.ParsedElement{
				Type: interfaces.ElementTypeStart,
				Name: t.Name.Local,
			}
			
			// Process attributes
			for _, attr := range t.Attr {
				if s.strictMode && !isValidAttribute(attr.Name.Local, attr.Value) {
					return fmt.Errorf("invalid attribute: %s", attr.Name.Local)
				}
				
				if element.Attributes == nil {
					element.Attributes = make(map[string]string)
				}
				element.Attributes[attr.Name.Local] = attr.Value
			}
			
			// Call callback
			if err := callback(element); err != nil {
				return err
			}
			
		case xml.EndElement:
			element := &interfaces.ParsedElement{
				Type: interfaces.ElementTypeEnd,
				Name: t.Name.Local,
			}
			
			// Call callback
			if err := callback(element); err != nil {
				return err
			}
			
		case xml.CharData:
			data := string(t)
			if len(strings.TrimSpace(data)) > 0 {
				element := &interfaces.ParsedElement{
					Type:    interfaces.ElementTypeCharData,
					Content: data,
				}
				
				// Call callback
				if err := callback(element); err != nil {
					return err
				}
			}
		}
	}
	
	return nil
}

// SetBufferSize sets the buffer size for streaming parsing.
func (s *streamParser) SetBufferSize(size int) {
	if size > 0 {
		s.bufferSize = size
	}
}

// GetBufferSize returns the current buffer size.
func (s *streamParser) GetBufferSize() int {
	return s.bufferSize
}

// SetMemoryLimit sets the memory limit for parsing large messages.
func (s *streamParser) SetMemoryLimit(limit int64) {
	atomic.StoreInt64(&s.maxMemory, limit)
}

// GetMemoryLimit returns the current memory limit.
func (s *streamParser) GetMemoryLimit() int64 {
	return atomic.LoadInt64(&s.maxMemory)
}

// GetCurrentMemoryUsage returns the current memory usage.
func (s *streamParser) GetCurrentMemoryUsage() int64 {
	return atomic.LoadInt64(&s.currentMemory)
}

// ResetMemoryUsage resets the memory usage counter.
func (s *streamParser) ResetMemoryUsage() {
	atomic.StoreInt64(&s.currentMemory, 0)
}

// SetStrictMode sets the parser to strict mode, which will enforce stricter validation.
func (s *streamParser) SetStrictMode(strict bool) {
	s.strictMode = strict
}

// GetStrictMode returns the current strict mode setting.
func (s *streamParser) GetStrictMode() bool {
	return s.strictMode
}

// updateMemoryUsage updates the memory usage counter based on the token.
func (s *streamParser) updateMemoryUsage(token xml.Token) {
	var size int64
	
	switch t := token.(type) {
	case xml.StartElement:
		// Estimate size of start element
		size += int64(len(t.Name.Local))
		for _, attr := range t.Attr {
			size += int64(len(attr.Name.Local) + len(attr.Value))
		}
	case xml.EndElement:
		// Estimate size of end element
		size += int64(len(t.Name.Local))
	case xml.CharData:
		// Estimate size of character data
		size += int64(len(t))
	case xml.Comment:
		// Estimate size of comment
		size += int64(len(t))
	case xml.ProcInst:
		// Estimate size of processing instruction
		size += int64(len(t.Target) + len(t.Inst))
	case xml.Directive:
		// Estimate size of directive
		size += int64(len(t))
	}
	
	// Add size to memory usage counter
	atomic.AddInt64(&s.currentMemory, size)
	
	// Check for real memory usage periodically
	// This is more expensive but more accurate
	if s.maxMemory > 0 && atomic.LoadInt64(&s.currentMemory) > s.maxMemory/2 {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		
		// Calculate memory growth since last check
		memoryGrowth := int64(m.Alloc)
		
		// If real memory usage is high, adjust our counter
		if memoryGrowth > s.maxMemory {
			// Log warning about high memory usage
			// logger.Warn("Memory usage approaching limit", "current", memoryGrowth, "limit", s.maxMemory)
			
			// If we're very close to the limit, force an early failure to prevent OOM
			if memoryGrowth > s.maxMemory*90/100 {
				atomic.StoreInt64(&s.currentMemory, s.maxMemory+1) // Force memory limit exceeded
				return
			}
			
			// Otherwise adjust our counter to better reflect real usage
			atomic.StoreInt64(&s.currentMemory, int64(float64(s.currentMemory)*1.5))
		}
	}
}

// isValidAttribute checks if an attribute is valid according to TR069 specifications.
func isValidAttribute(name, value string) bool {
	// This is a simplified implementation
	// In a real implementation, we would check against TR069 specifications
	
	// Check for empty name
	if name == "" {
		return false
	}
	
	// Check for invalid characters in name
	for _, c := range name {
		if !isValidNameChar(c) {
			return false
		}
	}
	
	return true
}

// isValidNameChar checks if a character is valid in an XML name.
func isValidNameChar(c rune) bool {
	// This is a simplified implementation
	// In a real implementation, we would check against XML specifications
	
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '_' || c == '-' || c == '.' || c == ':'
}