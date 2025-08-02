package protocol

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/unownone/shipitd/pkg/types"
	"github.com/sirupsen/logrus"
)

// Reader handles reading binary protocol messages from a connection
type Reader struct {
	conn   net.Conn
	reader *bufio.Reader
	logger *logrus.Logger
}

// NewReader creates a new protocol reader
func NewReader(conn net.Conn, logger *logrus.Logger) *Reader {
	return &Reader{
		conn:   conn,
		reader: bufio.NewReader(conn),
		logger: logger,
	}
}

// ReadMessage reads a complete message from the connection
func (r *Reader) ReadMessage() (*types.Message, error) {
	// Read message header (21 bytes: type(1) + tunnel_id(16) + payload_size(4))
	header := make([]byte, 21)
	_, err := io.ReadFull(r.reader, header)
	if err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("connection closed by peer")
		}
		return nil, fmt.Errorf("failed to read message header: %w", err)
	}

	// Parse message type
	msgType := types.MessageType(header[0])

	// Parse tunnel ID (16 bytes, trim null padding)
	tunnelID := string(header[1:17])
	tunnelID = string([]byte(tunnelID)[:len(tunnelID)-1]) // Remove null terminator

	// Parse payload size
	payloadSize := binary.BigEndian.Uint32(header[17:21])

	// Read payload
	var payload []byte
	if payloadSize > 0 {
		payload = make([]byte, payloadSize)
		_, err = io.ReadFull(r.reader, payload)
		if err != nil {
			return nil, fmt.Errorf("failed to read message payload: %w", err)
		}
	}

	// Create message
	message := &types.Message{
		Type:      msgType,
		TunnelID:  tunnelID,
		Payload:   payload,
		Timestamp: time.Now(),
	}

	r.logger.WithFields(logrus.Fields{
		"message_type": msgType,
		"tunnel_id":    tunnelID,
		"payload_size": payloadSize,
	}).Debug("Read message from connection")

	return message, nil
}

// ReadMessageWithTimeout reads a message with a timeout
func (r *Reader) ReadMessageWithTimeout(timeout time.Duration) (*types.Message, error) {
	// Set read deadline
	err := r.conn.SetReadDeadline(time.Now().Add(timeout))
	if err != nil {
		return nil, fmt.Errorf("failed to set read deadline: %w", err)
	}

	message, err := r.ReadMessage()
	if err != nil {
		return nil, err
	}

	// Clear read deadline
	err = r.conn.SetReadDeadline(time.Time{})
	if err != nil {
		r.logger.WithError(err).Warn("Failed to clear read deadline")
	}

	return message, nil
}

// ReadMessageAsync reads messages asynchronously and sends them to a channel
func (r *Reader) ReadMessageAsync(messageChan chan<- *types.Message, errorChan chan<- error) {
	defer func() {
		close(messageChan)
		close(errorChan)
	}()

	for {
		message, err := r.ReadMessage()
		if err != nil {
			if err.Error() == "connection closed by peer" {
				r.logger.Info("Connection closed by peer")
				return
			}
			r.logger.WithError(err).Error("Failed to read message")
			errorChan <- err
			return
		}

		messageChan <- message
	}
}

// Close closes the underlying connection
func (r *Reader) Close() error {
	return r.conn.Close()
}

// GetConnection returns the underlying connection
func (r *Reader) GetConnection() net.Conn {
	return r.conn
} 