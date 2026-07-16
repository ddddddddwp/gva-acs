package redact

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

const (
	connectionRequestPasswordName        = "Device.ManagementServer.ConnectionRequestPassword"
	connectionRequestPasswordPlaceholder = "__GVA_TR069_CONNECTION_REQUEST_PASSWORD__"
	legacyCiphertextPrefix               = "ciphertext-v"
	redactionMarker                      = "******"
)

type parameterValueState struct {
	depth      int
	nameDepth  int
	name       bytes.Buffer
	target     bool
	valueDepth int
}

// CWMPXML returns a sanitized copy of a CWMP payload. It redacts only the Value
// paired with an exact ConnectionRequestPassword Name inside a
// ParameterValueStruct. Malformed XML returns no payload.
func CWMPXML(in []byte) ([]byte, error) {
	if len(in) == 0 {
		return nil, nil
	}
	decoder := xml.NewDecoder(bytes.NewReader(in))
	var encoded bytes.Buffer
	encoder := xml.NewEncoder(&encoded)
	depth := 0
	redacted := false
	states := make([]parameterValueState, 0, 1)

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parse CWMP XML: %w", err)
		}

		switch typed := token.(type) {
		case xml.StartElement:
			depth++
			if typed.Name.Local == "ParameterValueStruct" {
				states = append(states, parameterValueState{depth: depth})
			}
			if len(states) > 0 {
				state := &states[len(states)-1]
				if depth == state.depth+1 && typed.Name.Local == "Name" {
					state.nameDepth = depth
					state.name.Reset()
				}
				if depth == state.depth+1 && typed.Name.Local == "Value" && state.target {
					state.valueDepth = depth
					redacted = true
					if err := encoder.EncodeToken(token); err != nil {
						return nil, fmt.Errorf("encode sanitized CWMP XML: %w", err)
					}
					if err := encoder.EncodeToken(xml.CharData(redactionMarker)); err != nil {
						return nil, fmt.Errorf("encode sanitized CWMP XML: %w", err)
					}
					continue
				}
			}
		case xml.CharData:
			if len(states) > 0 {
				state := &states[len(states)-1]
				if state.nameDepth > 0 {
					_, _ = state.name.Write(typed)
				}
				if state.valueDepth > 0 {
					continue
				}
			}
		case xml.EndElement:
			if len(states) > 0 {
				state := &states[len(states)-1]
				if state.nameDepth == depth && typed.Name.Local == "Name" {
					state.target = state.name.String() == connectionRequestPasswordName
					state.nameDepth = 0
				}
				if state.valueDepth == depth && typed.Name.Local == "Value" {
					state.valueDepth = 0
				}
			}
		}

		if err := encoder.EncodeToken(token); err != nil {
			return nil, fmt.Errorf("encode sanitized CWMP XML: %w", err)
		}
		if _, ok := token.(xml.EndElement); ok {
			if len(states) > 0 && states[len(states)-1].depth == depth {
				states = states[:len(states)-1]
			}
			depth--
		}
	}
	if err := encoder.Flush(); err != nil {
		return nil, fmt.Errorf("encode sanitized CWMP XML: %w", err)
	}
	if !redacted {
		return append([]byte(nil), in...), nil
	}
	return encoded.Bytes(), nil
}

// CommandJSON sanitizes structured command data without changing unrelated
// password fields. Internal persisted placeholders are never returned.
func CommandJSON(in []byte) ([]byte, error) {
	if len(in) == 0 {
		return nil, nil
	}
	value, err := decodeJSON(in)
	if err != nil {
		return nil, err
	}
	changed := false
	value = redactCommandJSONValue(value, &changed)
	if !changed {
		return append([]byte(nil), in...), nil
	}
	out, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode sanitized command JSON: %w", err)
	}
	return out, nil
}

// CommandText prevents internal credential representations from escaping in
// command-record metadata. Ordinary text, including unrelated password error
// messages, is returned unchanged.
func CommandText(in string) string {
	if strings.Contains(in, connectionRequestPasswordPlaceholder) || strings.Contains(in, legacyCiphertextPrefix) {
		return redactionMarker
	}
	return in
}

// ConnectionProfileJSON sanitizes the write-only password in a connection
// profile request. Nested or otherwise unrelated password fields are retained.
func ConnectionProfileJSON(in []byte) ([]byte, error) {
	if len(in) == 0 {
		return nil, nil
	}
	value, err := decodeJSON(in)
	if err != nil {
		return nil, err
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("connection profile JSON must be an object")
	}
	if _, exists := object["password"]; !exists {
		return append([]byte(nil), in...), nil
	}
	object["password"] = redactionMarker
	out, err := json.Marshal(object)
	if err != nil {
		return nil, fmt.Errorf("encode sanitized connection profile JSON: %w", err)
	}
	return out, nil
}

func decodeJSON(in []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(in))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("parse protected JSON: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("parse protected JSON: multiple documents")
		}
		return nil, fmt.Errorf("parse protected JSON: %w", err)
	}
	return value, nil
}

func redactCommandJSONValue(value any, changed *bool) any {
	switch typed := value.(type) {
	case string:
		sanitized := CommandText(typed)
		if sanitized != typed {
			*changed = true
		}
		return sanitized
	case map[string]any:
		redactParameterValue(typed, "name", "value", changed)
		redactParameterValue(typed, "Name", "Value", changed)
		for key, child := range typed {
			if isLegacyCiphertextKey(key) {
				typed[key] = redactionMarker
				*changed = true
				continue
			}
			typed[key] = redactCommandJSONValue(child, changed)
		}
	case []any:
		for index, child := range typed {
			typed[index] = redactCommandJSONValue(child, changed)
		}
	}
	return value
}

func isLegacyCiphertextKey(key string) bool {
	switch key {
	case "passwordCiphertext", "password_ciphertext", "credentialCiphertext", "credential_ciphertext":
		return true
	default:
		return false
	}
}

func redactParameterValue(object map[string]any, nameKey, valueKey string, changed *bool) {
	name, ok := object[nameKey].(string)
	if !ok || name != connectionRequestPasswordName {
		return
	}
	if _, exists := object[valueKey]; exists {
		object[valueKey] = redactionMarker
		*changed = true
	}
}
