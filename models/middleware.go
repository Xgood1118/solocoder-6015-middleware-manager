
package models

import (
	"encoding/json"
	"time"
)

// Middleware represents a Traefik middleware configuration
type Middleware struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Config    string    `json:"config"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ConfigMap returns the middleware config as a map
func (m *Middleware) ConfigMap() (map[string]interface{}, error) {
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(m.Config), &config); err != nil {
		return nil, err
	}
	return config, nil
}

// ResourceMiddleware represents the relationship between a resource and a middleware
type ResourceMiddleware struct {
	ResourceID   string    `json:"resource_id"`
	MiddlewareID string    `json:"middleware_id"`
	Priority     int       `json:"priority"`
	CreatedAt    time.Time `json:"created_at"`
}
// Resource struct removed to resolve redeclaration error.
// Please ensure the Resource struct is only defined in one file (likely resource.go).