package protocol

import (
	"fmt"
	"net"
	"time"

	"github.com/unownone/shipitd/pkg/types"
	"github.com/sirupsen/logrus"
)

// Writer handles writing binary protocol messages to a connection
type Writer struct {
	conn   net.Conn
	logger *logrus.Logger
}

// NewWriter creates a new protocol writer
func NewWriter(conn net.Conn, logger *logrus.Logger) *Writer {
	return &Writer{
		conn:   conn,
		logger: logger,
	}
}

// WriteMessage writes a message to the connection
func (w *Writer) WriteMessage(message *types.Message) error {
	// Serialize the message
	data, err := message.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize message: %w", err)
	}

	// Write the message
	_, err = w.conn.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	w.logger.WithFields(logrus.Fields{
		"message_type": message.Type,
		"tunnel_id":    message.TunnelID,
		"payload_size": len(message.Payload),
	}).Debug("Wrote message to connection")

	return nil
}

// WriteMessageWithTimeout writes a message with a timeout
func (w *Writer) WriteMessageWithTimeout(message *types.Message, timeout time.Duration) error {
	// Set write deadline
	err := w.conn.SetWriteDeadline(time.Now().Add(timeout))
	if err != nil {
		return fmt.Errorf("failed to set write deadline: %w", err)
	}

	err = w.WriteMessage(message)
	if err != nil {
		return err
	}

	// Clear write deadline
	err = w.conn.SetWriteDeadline(time.Time{})
	if err != nil {
		w.logger.WithError(err).Warn("Failed to clear write deadline")
	}

	return nil
}

// WriteTunnelRegistration writes a tunnel registration message
func (w *Writer) WriteTunnelRegistration(tunnelID string, payload *types.TunnelRegistrationPayload) error {
	message, err := types.NewTunnelRegistrationMessage(tunnelID, payload)
	if err != nil {
		return fmt.Errorf("failed to create tunnel registration message: %w", err)
	}

	return w.WriteMessage(message)
}

// WriteDataResponse writes a data response message
func (w *Writer) WriteDataResponse(tunnelID string, payload *types.DataResponsePayload) error {
	message, err := types.NewDataResponseMessage(tunnelID, payload)
	if err != nil {
		return fmt.Errorf("failed to create data response message: %w", err)
	}

	return w.WriteMessage(message)
}

// WriteHeartbeat writes a heartbeat message
func (w *Writer) WriteHeartbeat(tunnelID string, payload *types.HeartbeatPayload) error {
	message, err := types.NewHeartbeatMessage(tunnelID, payload)
	if err != nil {
		return fmt.Errorf("failed to create heartbeat message: %w", err)
	}

	return w.WriteMessage(message)
}

// WriteError writes an error message
func (w *Writer) WriteError(tunnelID string, payload *types.ErrorPayload) error {
	message, err := types.NewErrorMessage(tunnelID, payload)
	if err != nil {
		return fmt.Errorf("failed to create error message: %w", err)
	}

	return w.WriteMessage(message)
}

// WriteAcknowledge writes an acknowledgment message
func (w *Writer) WriteAcknowledge(tunnelID string, payload *types.AcknowledgePayload) error {
	message, err := types.NewAcknowledgeMessage(tunnelID, payload)
	if err != nil {
		return fmt.Errorf("failed to create acknowledgment message: %w", err)
	}

	return w.WriteMessage(message)
}

// WriteConnectionClose writes a connection close message
func (w *Writer) WriteConnectionClose(tunnelID string, payload *types.ConnectionClosePayload) error {
	message, err := types.NewConnectionCloseMessage(tunnelID, payload)
	if err != nil {
		return fmt.Errorf("failed to create connection close message: %w", err)
	}

	return w.WriteMessage(message)
}

// Close closes the underlying connection
func (w *Writer) Close() error {
	return w.conn.Close()
}

// GetConnection returns the underlying connection
func (w *Writer) GetConnection() net.Conn {
	return w.conn
} 