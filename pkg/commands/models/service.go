package models

// ServiceStatus represents the running status of a service.
type ServiceStatus string

const (
	ServiceRunning ServiceStatus = "started"
	ServiceStopped ServiceStatus = "stopped"
	ServiceError   ServiceStatus = "error"
	ServiceNone    ServiceStatus = "none"
	ServiceUnknown ServiceStatus = "unknown"
)

// Service represents a Homebrew-managed service.
type Service struct {
	Name     string
	Status   ServiceStatus
	User     string
	File     string
	ExitCode int
}

// IsRunning returns true if the service is currently running.
func (s *Service) IsRunning() bool {
	return s.Status == ServiceRunning
}

// StatusIcon returns a display icon for the service status.
func (s *Service) StatusIcon() string {
	switch s.Status {
	case ServiceRunning:
		return "●"
	case ServiceStopped:
		return "■"
	case ServiceError:
		return "✖"
	default:
		return "○"
	}
}
