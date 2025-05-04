package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"time"
)

var playerVar string
var reinstallVar bool

func init() {
	flag.StringVar(&playerVar, "player", "", "deeplink or roblox-player protocol")
	flag.BoolVar(&reinstallVar, "reinstall", false, "whether to reinstall the client")
}

func main() {
	// parse cli args
	flag.Parse()
	configDir := getConfigDirectory()

	var logWriter io.Writer = os.Stderr
	logDir := path.Join(configDir, "logs")

	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Could not create log directory %s: %v. Logging to stderr.\n", logDir, err)
		log.SetOutput(logWriter)
	} else {
		logFilePath := path.Join(logDir, fmt.Sprintf("ll-%s.log.txt", time.Now().Format("2006-01-02T15-04-05")))
		logFile, err := os.OpenFile(logFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not open log file %s: %v. Logging to stderr.\n", logFilePath, err)
		} else {
			logWriter = io.MultiWriter(os.Stderr, logFile)
			defer logFile.Close()

			log.SetOutput(logWriter)

			log.Printf("Logging to file: %s", logFilePath)
		}
	}

	selfPath, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to locate executable path: %s", err)
	}

	log.Printf("Leeklaunch started from %s", selfPath)

	log.Println("Attempting to (re)register Roblox protocols")
	if err := registerProtocols(selfPath); err != nil {
		log.Fatalf("failed to register roblox protocols: %s", err)
	}

	if wasFlagPassed("reinstall") && reinstallVar {
		log.Println("Reinstalling client")

		if err := saveDeployment(); err != nil {
			log.Fatalf("failed to save deployment: %s", err)
		}
	}

	if wasFlagPassed("player") {
		log.Printf("Launching Roblox with deeplink: %s", playerVar)
		if err := launchRoblox(playerVar); err != nil {
			log.Fatalf("failed to launch roblox: %s", err)
		}
	}
}
