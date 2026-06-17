package models

import (
	"encoding/json"
	"time"
)

// Service represents a Traefik service configuration
type Service struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Config    string    `json:"config"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ServiceType represents valid service types
type ServiceType string

const (
	LoadBalancerType ServiceType = "loadBalancer"
	WeightedType     ServiceType = "weighted"
	MirroringType    ServiceType = "mirroring"
	FailoverType     ServiceType = "failover"
)

// IsValidServiceType checks if a service type is valid
func IsValidServiceType(typ string) bool {
	validTypes := map[string]bool{
		string(LoadBalancerType): true,
		string(WeightedType):     true,
		string(MirroringType):    true,
		string(FailoverType):     true,
	}
	return validTypes[typ]
}

// ConfigMap returns the service config as a map
func (s *Service) ConfigMap() (map[string]interface{}, error) {
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(s.Config), &config); err != nil {
		return nil, err
	}
	return config, nil
}

// ResourceService represents the relationship between a resource and a service
type ResourceService struct {
	ResourceID string    `json:"resource_id"`
	ServiceID  string    `json:"service_id"`
	CreatedAt  time.Time `json:"created_at"`
}

type ServiceCollection struct {
    Services []Service `json:"services"`
}


// LoadBalancerConfig represents the configuration for a LoadBalancer service
type LoadBalancerConfig struct {
	Servers []ServerConfig `json:"servers"`
	// Optional configurations
	PassHostHeader bool `json:"passHostHeader,omitempty"`
	
	// Health check configuration
	HealthCheck *HealthCheckConfig `json:"healthCheck,omitempty"`
	
	// Sticky sessions
	Sticky *StickyConfig `json:"sticky,omitempty"`
	
	// Response forwarding
	ResponseForwarding *ResponseForwardingConfig `json:"responseForwarding,omitempty"`
	
	// Servers transport
	ServersTransport string `json:"serversTransport,omitempty"`
}

// ServerConfig represents a server in a LoadBalancer
type ServerConfig struct {
	URL       string `json:"url"`
	Weight    *int   `json:"weight,omitempty"`
	TLS       *bool  `json:"tls,omitempty"`
	Address   string `json:"address,omitempty"` // For TCP/UDP services
}

// HealthCheckConfig represents health check configuration
type HealthCheckConfig struct {
	Path     string            `json:"path,omitempty"`
	Port     *int              `json:"port,omitempty"`
	Interval string            `json:"interval,omitempty"`
	Timeout  string            `json:"timeout,omitempty"`
	Scheme   string            `json:"scheme,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
}

// StickyConfig represents sticky session configuration
type StickyConfig struct {
	Cookie *CookieConfig `json:"cookie,omitempty"`
}

// CookieConfig represents cookie configuration for sticky sessions
type CookieConfig struct {
	Name     string `json:"name,omitempty"`
	Secure   bool   `json:"secure,omitempty"`
	HTTPOnly bool   `json:"httpOnly,omitempty"`
}

// ResponseForwardingConfig represents response forwarding configuration
type ResponseForwardingConfig struct {
	FlushInterval string `json:"flushInterval,omitempty"`
}

// WeightedConfig represents the configuration for a Weighted service
type WeightedConfig struct {
	Services []WeightedServiceConfig `json:"services"`
	// Optional configuration
	Sticky      *StickyConfig      `json:"sticky,omitempty"`
	HealthCheck *HealthCheckConfig `json:"healthCheck,omitempty"`
}

// WeightedServiceConfig represents a service in a weighted group
type WeightedServiceConfig struct {
	Name   string `json:"name"`
	Weight int    `json:"weight"`
}

// MirroringConfig represents the configuration for a Mirroring service
type MirroringConfig struct {
	Service    string                `json:"service"`
	Mirrors    []MirrorServiceConfig `json:"mirrors"`
	// Optional configuration
	MirrorBody *bool                `json:"mirrorBody,omitempty"`
	MaxBodySize *int                `json:"maxBodySize,omitempty"`
	HealthCheck *HealthCheckConfig  `json:"healthCheck,omitempty"`
}

// MirrorServiceConfig represents a service in a mirroring setup
type MirrorServiceConfig struct {
	Name    string `json:"name"`
	Percent int    `json:"percent"`
}

// FailoverConfig represents the configuration for a Failover service
type FailoverConfig struct {
	Service     string           `json:"service"`
	Fallback    string           `json:"fallback"`
	// Optional configuration
	HealthCheck *HealthCheckConfig `json:"healthCheck,omitempty"`
}

// ServiceProcessor handles type-specific processing for different service types
type ServiceProcessor interface {
	Process(config map[string]interface{}) map[string]interface{}
}

// DefaultServiceProcessor is the default handler for service configurations
type DefaultServiceProcessor struct{}

// Process handles general service configuration processing
func (p *DefaultServiceProcessor) Process(config map[string]interface{}) map[string]interface{} {
	return preserveTraefikValues(config).(map[string]interface{})
}

// GetServiceProcessor returns the appropriate processor for a service type
func GetServiceProcessor(serviceType string) ServiceProcessor {
	// For now, we use the default processor for all service types
	// In the future, we could have specialized processors for different service types
	return &DefaultServiceProcessor{}
}

// ProcessServiceConfig processes a service configuration based on its type
func ProcessServiceConfig(serviceType string, config map[string]interface{}) map[string]interface{} {
	processor := GetServiceProcessor(serviceType)
	return processor.Process(config)
}