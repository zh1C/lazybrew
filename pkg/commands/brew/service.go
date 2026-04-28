package brew

import (
	"encoding/json"
	"fmt"

	"github.com/zh1C/lazybrew/pkg/commands/models"
)

// ServiceCommands provides operations for Homebrew services.
type ServiceCommands struct {
	runner *Runner
}

// NewServiceCommands creates a new ServiceCommands instance.
func NewServiceCommands(runner *Runner) *ServiceCommands {
	return &ServiceCommands{runner: runner}
}

type jsonService struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	User     string `json:"user"`
	File     string `json:"file"`
	ExitCode int    `json:"exit_code"`
}

// List returns all managed services.
func (sc *ServiceCommands) List() ([]models.Service, error) {
	result := sc.runner.Run("services", "list", "--json")
	if result.Err != nil {
		return nil, fmt.Errorf("failed to list services: %w", result.Err)
	}

	var jsonServices []jsonService
	if err := json.Unmarshal([]byte(result.Stdout), &jsonServices); err != nil {
		return nil, fmt.Errorf("failed to parse services JSON: %w", err)
	}

	services := make([]models.Service, 0, len(jsonServices))
	for _, js := range jsonServices {
		s := models.Service{
			Name:     js.Name,
			Status:   parseServiceStatus(js.Status),
			User:     js.User,
			File:     js.File,
			ExitCode: js.ExitCode,
		}
		services = append(services, s)
	}
	return services, nil
}

// Start starts a service.
func (sc *ServiceCommands) Start(name string, onOutput func(string)) CommandResult {
	return sc.runner.RunWithCallback(onOutput, "services", "start", name)
}

// Stop stops a service.
func (sc *ServiceCommands) Stop(name string, onOutput func(string)) CommandResult {
	return sc.runner.RunWithCallback(onOutput, "services", "stop", name)
}

// Restart restarts a service.
func (sc *ServiceCommands) Restart(name string, onOutput func(string)) CommandResult {
	return sc.runner.RunWithCallback(onOutput, "services", "restart", name)
}

func parseServiceStatus(s string) models.ServiceStatus {
	switch s {
	case "started":
		return models.ServiceRunning
	case "stopped":
		return models.ServiceStopped
	case "error":
		return models.ServiceError
	case "none":
		return models.ServiceNone
	default:
		return models.ServiceUnknown
	}
}
