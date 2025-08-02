package types

import (
	"time"
)

// Tunnel represents a tunnel configuration
type Tunnel struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	LocalURL    string    `json:"local_url"`
	Protocol    string    `json:"protocol"`
	Description string    `json:"description"`
	PublicURL   string    `json:"public_url"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TunnelRequest represents a tunnel creation request
type TunnelRequest struct {
	Name        string `json:"name"`
	LocalURL    string `json:"local_url"`
	Protocol    string `json:"protocol"`
	Description string `json:"description"`
}

// TunnelResponse represents a tunnel response
type TunnelResponse struct {
	Tunnel *Tunnel `json:"tunnel"`
}

// TunnelsResponse represents a list of tunnels response
type TunnelsResponse struct {
	Tunnels []*Tunnel `json:"tunnels"`
	Count   int       `json:"count"`
} 