package doctor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/vsyaco/kidney/internal/domain"
	"github.com/vsyaco/kidney/internal/transport"
)

type Check struct {
	Name     string `json:"name"`
	OK       bool   `json:"ok"`
	Message  string `json:"message"`
	Remedy   string `json:"remedy,omitempty"`
	Critical bool   `json:"critical"`
}

type Report struct {
	Checks  []Check         `json:"checks"`
	Devices []domain.Device `json:"devices"`
}

func Run(ctx context.Context, transports []domain.Transport) Report {
	checks := []Check{
		{
			Name:    "direct MTP backend",
			OK:      true,
			Message: "built into Kidney; no simple-mtpfs or macFUSE required",
		},
	}

	devices := make([]domain.Device, 0)
	for _, item := range transports {
		detected, err := item.Detect(ctx)
		if err != nil {
			checks = append(checks, Check{
				Name:     item.Name() + " detection",
				OK:       false,
				Message:  err.Error(),
				Critical: false,
			})
			continue
		}

		devices = append(devices, detected...)
		checks = append(checks, Check{
			Name:     item.Name() + " detection",
			OK:       len(detected) > 0,
			Message:  detectionMessage(item.Name(), len(detected)),
			Critical: false,
		})
	}

	return Report{
		Checks:  checks,
		Devices: devices,
	}
}

func Print(report Report) string {
	var builder strings.Builder

	builder.WriteString("Kidney doctor\n\n")
	for _, check := range report.Checks {
		status := "ok"
		if !check.OK {
			status = "missing"
		}

		builder.WriteString(fmt.Sprintf("[%s] %s: %s\n", status, check.Name, check.Message))
		if !check.OK && check.Remedy != "" {
			builder.WriteString(fmt.Sprintf("      %s\n", check.Remedy))
		}
	}

	builder.WriteString("\nDevices:\n")
	if len(report.Devices) == 0 {
		builder.WriteString("  none\n")
		return builder.String()
	}

	for _, device := range report.Devices {
		builder.WriteString(fmt.Sprintf(
			"  - %s (%s) backend=%s documents=%s serial=%s %s\n",
			device.Name,
			device.ID,
			device.Backend,
			device.DocumentsPath,
			device.Serial,
			device.Message,
		))
	}

	return builder.String()
}

func detectionMessage(name string, count int) string {
	if count == 0 {
		if name == "mtp" {
			return "no Kindle MTP device found; unlock the Kindle and choose USB file transfer if prompted"
		}
		return "no mounted Kindle-like disk volume found"
	}

	return fmt.Sprintf("%d device(s) found", count)
}

func Context(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, 20*time.Second)
}

func SupportedExtensionsLine() string {
	return strings.Join(transport.SupportedExtensions(), ", ")
}
