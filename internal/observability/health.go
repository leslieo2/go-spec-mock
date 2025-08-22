package observability

import "time"

type HealthStatus struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Version   string                 `json:"version"`
	Uptime    string                 `json:"uptime"`
	Metrics   map[string]interface{} `json:"metrics,omitempty"`
	Checks    map[string]bool        `json:"checks"`
}
