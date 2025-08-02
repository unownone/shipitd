package types

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"
)

// MessageType represents the type of protocol message
type MessageType uint8

const (
	// MessageTypeTunnelRegistration represents a tunnel registration message
	MessageTypeTunnelRegistration MessageType = 0x01
	// MessageTypeDataForward represents data forwarded from server to client
	MessageTypeDataForward MessageType = 0x02
	// MessageTypeDataResponse represents response data from client to server
	MessageTypeDataResponse MessageType = 0x03
	// MessageTypeConnectionClose represents a connection close message
	MessageTypeConnectionClose MessageType = 0x04
	// MessageTypeHeartbeat represents a heartbeat message
	MessageTypeHeartbeat MessageType = 0x05
	// MessageTypeError represents an error message
	MessageTypeError MessageType = 0x06
	// MessageTypeAcknowledge represents an acknowledgment message
	MessageTypeAcknowledge MessageType = 0x07
)

// Message represents a protocol message
type Message struct {
	Type      MessageType `json:"type"`
	TunnelID  string     `json:"tunnel_id"`
	Payload   []byte     `json:"payload"`
	Timestamp time.Time  `json:"timestamp"`
}

// TunnelRegistrationPayload represents tunnel registration data
type TunnelRegistrationPayload struct {
	Protocol      string `json:"protocol"`
	LocalPort     int    `json:"local_port"`
	Subdomain     string `json:"subdomain,omitempty"`
	PublicPort    *int   `json:"public_port,omitempty"`
	MaxConnections int   `json:"max_connections"`
}

// DataForwardPayload represents data forwarded from server
type DataForwardPayload struct {
	ConnectionID string            `json:"connection_id"`
	RequestID    string            `json:"request_id"`
	Data         []byte            `json:"data"`
	Headers      map[string]string `json:"headers"`
	Method       string            `json:"method"`
	Path         string            `json:"path"`
}

// DataResponsePayload represents response data from client
type DataResponsePayload struct {
	ConnectionID string            `json:"connection_id"`
	RequestID    string            `json:"request_id"`
	Data         []byte            `json:"data"`
	StatusCode   int               `json:"status_code"`
	Headers      map[string]string `json:"headers"`
}

// HeartbeatPayload represents heartbeat data
type HeartbeatPayload struct {
	Timestamp    int64 `json:"timestamp"`
	ActiveConns  int   `json:"active_conns"`
	TotalRequests int  `json:"total_requests"`
}

// ErrorPayload represents error data
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// AcknowledgePayload represents acknowledgment data
type AcknowledgePayload struct {
	MessageID string `json:"message_id"`
	Status    string `json:"status"`
}

// ConnectionClosePayload represents connection close data
type ConnectionClosePayload struct {
	ConnectionID string `json:"connection_id"`
	Reason       string `json:"reason"`
}

// Serialize serializes a message to binary format
func (m *Message) Serialize() ([]byte, error) {
	// Calculate total size
	payloadSize := len(m.Payload)
	totalSize := 1 + 16 + 4 + payloadSize // type + tunnel_id + payload_size + payload
	
	// Create buffer
	buf := make([]byte, totalSize)
	offset := 0
	
	// Write message type (1 byte)
	buf[offset] = uint8(m.Type)
	offset++
	
	// Write tunnel ID (16 bytes, padded with zeros)
	tunnelIDBytes := []byte(m.TunnelID)
	if len(tunnelIDBytes) > 16 {
		tunnelIDBytes = tunnelIDBytes[:16]
	}
	copy(buf[offset:offset+16], tunnelIDBytes)
	offset += 16
	
	// Write payload size (4 bytes)
	binary.BigEndian.PutUint32(buf[offset:offset+4], uint32(payloadSize))
	offset += 4
	
	// Write payload
	copy(buf[offset:], m.Payload)
	
	return buf, nil
}

// Deserialize deserializes a message from binary format
func (m *Message) Deserialize(data []byte) error {
	if len(data) < 21 { // minimum size: type(1) + tunnel_id(16) + payload_size(4)
		return fmt.Errorf("message too short: %d bytes", len(data))
	}
	
	offset := 0
	
	// Read message type
	m.Type = MessageType(data[offset])
	offset++
	
	// Read tunnel ID
	tunnelIDBytes := data[offset : offset+16]
	m.TunnelID = string(bytes.TrimRight(tunnelIDBytes, "\x00"))
	offset += 16
	
	// Read payload size
	payloadSize := binary.BigEndian.Uint32(data[offset : offset+4])
	offset += 4
	
	// Read payload
	if len(data) < offset+int(payloadSize) {
		return fmt.Errorf("message truncated: expected %d bytes, got %d", offset+int(payloadSize), len(data))
	}
	m.Payload = data[offset : offset+int(payloadSize)]
	
	return nil
}

// NewMessage creates a new message
func NewMessage(msgType MessageType, tunnelID string, payload []byte) *Message {
	return &Message{
		Type:      msgType,
		TunnelID:  tunnelID,
		Payload:   payload,
		Timestamp: time.Now(),
	}
}

// NewTunnelRegistrationMessage creates a new tunnel registration message
func NewTunnelRegistrationMessage(tunnelID string, payload *TunnelRegistrationPayload) (*Message, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return NewMessage(MessageTypeTunnelRegistration, tunnelID, data), nil
}

// NewDataForwardMessage creates a new data forward message
func NewDataForwardMessage(tunnelID string, payload *DataForwardPayload) (*Message, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return NewMessage(MessageTypeDataForward, tunnelID, data), nil
}

// NewDataResponseMessage creates a new data response message
func NewDataResponseMessage(tunnelID string, payload *DataResponsePayload) (*Message, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return NewMessage(MessageTypeDataResponse, tunnelID, data), nil
}

// NewHeartbeatMessage creates a new heartbeat message
func NewHeartbeatMessage(tunnelID string, payload *HeartbeatPayload) (*Message, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return NewMessage(MessageTypeHeartbeat, tunnelID, data), nil
}

// NewErrorMessage creates a new error message
func NewErrorMessage(tunnelID string, payload *ErrorPayload) (*Message, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return NewMessage(MessageTypeError, tunnelID, data), nil
}

// NewAcknowledgeMessage creates a new acknowledgment message
func NewAcknowledgeMessage(tunnelID string, payload *AcknowledgePayload) (*Message, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return NewMessage(MessageTypeAcknowledge, tunnelID, data), nil
}

// NewConnectionCloseMessage creates a new connection close message
func NewConnectionCloseMessage(tunnelID string, payload *ConnectionClosePayload) (*Message, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return NewMessage(MessageTypeConnectionClose, tunnelID, data), nil
}

// ParsePayload parses the message payload based on message type
func (m *Message) ParsePayload() (interface{}, error) {
	switch m.Type {
	case MessageTypeTunnelRegistration:
		var payload TunnelRegistrationPayload
		err := json.Unmarshal(m.Payload, &payload)
		return &payload, err
	case MessageTypeDataForward:
		var payload DataForwardPayload
		err := json.Unmarshal(m.Payload, &payload)
		return &payload, err
	case MessageTypeDataResponse:
		var payload DataResponsePayload
		err := json.Unmarshal(m.Payload, &payload)
		return &payload, err
	case MessageTypeHeartbeat:
		var payload HeartbeatPayload
		err := json.Unmarshal(m.Payload, &payload)
		return &payload, err
	case MessageTypeError:
		var payload ErrorPayload
		err := json.Unmarshal(m.Payload, &payload)
		return &payload, err
	case MessageTypeAcknowledge:
		var payload AcknowledgePayload
		err := json.Unmarshal(m.Payload, &payload)
		return &payload, err
	case MessageTypeConnectionClose:
		var payload ConnectionClosePayload
		err := json.Unmarshal(m.Payload, &payload)
		return &payload, err
	default:
		return nil, fmt.Errorf("unknown message type: %d", m.Type)
	}
}

// String returns a string representation of the message
func (m *Message) String() string {
	return fmt.Sprintf("Message{Type: %d, TunnelID: %s, PayloadSize: %d, Timestamp: %s}",
		m.Type, m.TunnelID, len(m.Payload), m.Timestamp.Format(time.RFC3339))
} 