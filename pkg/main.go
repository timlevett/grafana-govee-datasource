package main

import (
	"os"

	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"

	"github.com/timlevett/grafana-govee-datasource/pkg/plugin"
)

func main() {
	// Start the plugin. datasource.Manage will block until the plugin exits or
	// Grafana sends a shutdown signal.
	if err := datasource.Manage(
		"timlevett-govee-datasource",
		plugin.NewGoveeDatasource,
		datasource.ManageOpts{},
	); err != nil {
		log.DefaultLogger.Error("Plugin exited with error", "error", err.Error())
		os.Exit(1)
	}
}
