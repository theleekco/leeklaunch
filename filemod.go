package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

/*
probably should compare hashes of the files in mods dir and the client dir

but i need to get this working & pushed to repo fast lol
*/

// Replaces files in the client directory with those from the mods directory
func patchFiles() error {
	clientPath, err := getLatestVersionSave()
	leekConfig := getConfigDirectory()

	if err != nil {
		return fmt.Errorf("failed to get latest version save: %v", err)
	}

	modPath := filepath.Join(leekConfig, "mods")
	if _, err := os.Stat(modPath); os.IsNotExist(err) {
		log.Printf("No mods directory found at %s, skipping file patching", modPath)
		return nil
	}

	err = filepath.Walk(modPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing %s: %v", filePath, err)
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(modPath, filePath)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %v", filePath, err)
		}

		clientFilePath := filepath.Join(clientPath, relPath)

		if _, err := os.Stat(clientFilePath); os.IsNotExist(err) {
			log.Printf("File %s does not exist", relPath)
			return nil
		} else {
			sourceData, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to read source file %s: %v", filePath, err)
			}

			err = os.WriteFile(clientFilePath, sourceData, 0644)
			if err != nil {
				return fmt.Errorf("failed to write to destination file %s: %v", clientFilePath, err)
			}

			log.Printf("Replacing asset %s", relPath)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk mods directory: %v", err)
	}

	log.Println("Done replacing assets from mods directory")
	return nil
}
