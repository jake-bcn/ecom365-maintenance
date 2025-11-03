package ecom365_maintenance

import (
	"embed"
)

//go:embed templates/*
var content embed.FS

var maintenancePage string

func getMaintenancePage() string {
	if maintenancePage != "" {
		return maintenancePage
	}

	// 方法一：使用 embed.FS 的 ReadFile 方法
	templateData, err := content.ReadFile("templates/e-com365_503_3.html")
	if err != nil {
		maintenancePage = "Our site is in maintenance mode"
		return maintenancePage
	}

	maintenancePage = string(templateData)
	return maintenancePage
}
