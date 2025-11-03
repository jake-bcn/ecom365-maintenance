package ecom365_maintenance

import (
	"embed"
	"io/fs"
)

//go:embed templates/*
var content embed.FS

var maintenancePage string

func getMaintencePage() string {
	if maintenancePage != "" {
		return maintenancePage
	}
	templateData, err := fs.ReadFile(content, "templates/e-com365_503_3.html")
	if err != nil {
		maintenancePage = "Our site is in maintenance mode"
		return maintenancePage
	}
	maintenancePage = string(templateData)
	return maintenancePage
}
