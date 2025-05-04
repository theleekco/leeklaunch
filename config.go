package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

func registerProtocols(execPath string) error {
	paths := []string{
		"roblox-player",
		"roblox",
	}

	for _, path := range paths {
		key, err := registry.OpenKey(registry.CLASSES_ROOT, path+"\\shell\\open\\command", registry.QUERY_VALUE|registry.SET_VALUE)

		if err != nil {
			return fmt.Errorf("failed to read registry key: %v", err)
		}

		regValue := fmt.Sprintf("\"%s\" -player \"%s\"", execPath, "%1")

		err = key.SetStringValue("", regValue)

		if err != nil {
			return fmt.Errorf("failed to write registry key: %v", err)
		}
		defer key.Close()
	}

	return nil
}

func getConfigDirectory() string {
	userConfig, err := os.UserConfigDir()
	if err != nil {
		log.Fatalf("failed to get user config directory: %v", err)
	}

	configDir := filepath.Join(userConfig, "leeklaunch")

	err = os.MkdirAll(configDir, 0755)
	if err != nil {
		log.Fatalf("failed to get user config directory: %v", err)
	}

	return configDir
}

func wasFlagPassed(flagName string) bool {
	flagPassed := false

	flag.Visit(func(f *flag.Flag) {
		if f.Name == flagName {
			flagPassed = true
			return
		}
	})

	return flagPassed
}
