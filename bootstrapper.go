package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

type ClientSettingsResponse struct {
	Version             string
	ClientVersionUpload string
	BootstrapperVersion string
}

const appsettings = `<Settings>
<ContentFolder>content</ContentFolder>
<BaseUrl>http://www.roblox.com</BaseUrl>
</Settings>`

var EXTRACTION_ROOTS = map[string]string{
	"RobloxApp.zip":                     "./",
	"redist.zip":                        "./",
	"shaders.zip":                       "./shaders",
	"ssl.zip":                           "./ssl",
	"WebView2.zip":                      "./",
	"WebView2RuntimeInstaller.zip":      "./WebView2RuntimeInstaller",
	"content-avatar.zip":                "./content/avatar",
	"content-configs.zip":               "./content/configs",
	"content-fonts.zip":                 "./content/fonts",
	"content-sky.zip":                   "./content/sky",
	"content-sounds.zip":                "./content/sounds",
	"content-textures2.zip":             "./content/textures",
	"content-models.zip":                "./content/models",
	"content-platform-fonts.zip":        "./PlatformContent/pc/fonts",
	"content-platform-dictionaries.zip": "./PlatformContent/pc/shared_compression_dictionaries",
	"content-terrain.zip":               "./PlatformContent/pc/terrain",
	"content-textures3.zip":             "./PlatformContent/pc/textures",
	"extracontent-places.zip":           "./ExtraContent/places",
	"extracontent-luapackages.zip":      "./ExtraContent/LuaPackages",
	"extracontent-translations.zip":     "./ExtraContent/translations",
	"extracontent-models.zip":           "./ExtraContent/models",
	"extracontent-textures.zip":         "./ExtraContent/textures",
}

func getClientInfo() (ClientSettingsResponse, error) {
	channel := "LIVE" // replace later
	clientSettingsURL := fmt.Sprintf("https://clientsettingscdn.roblox.com/v2/client-version/WindowsPlayer/channel/%s", channel)

	response, err := http.Get(clientSettingsURL)
	if err != nil {
		return ClientSettingsResponse{}, fmt.Errorf("failed to get client settings: %v", err)
	}
	defer response.Body.Close()

	var clientInfo ClientSettingsResponse
	if err := json.NewDecoder(response.Body).Decode(&clientInfo); err != nil {
		return ClientSettingsResponse{}, fmt.Errorf("failed to decode client settings JSON: %v", err)
	}

	return clientInfo, nil
}

func getVersionManifest() (string, error) {
	clientInfo, err := getClientInfo()
	if err != nil {
		return "", fmt.Errorf("failed to get client info: %v", err)
	}

	manifestURL := fmt.Sprintf("https://setup.rbxcdn.com/%s-rbxPkgManifest.txt", clientInfo.ClientVersionUpload)

	response, err := http.Get(manifestURL)
	if err != nil {
		return "", fmt.Errorf("failed to get version manifest: %v", err)
	}
	defer response.Body.Close()

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	return string(bodyBytes), nil
}

func getDownloadableArchives(manifest string) ([]string, error) {
	const expectedManifestVersion = "v0"

	lines := strings.Split(manifest, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != expectedManifestVersion {
		return nil, fmt.Errorf("unexpected manifest version: %s", lines[0])
	}

	var archives []string
	for _, line := range lines[1:] {
		trimmedLine := strings.TrimSpace(line)
		if strings.HasSuffix(trimmedLine, ".zip") {
			archives = append(archives, trimmedLine)
		}
	}

	return archives, nil
}

func saveDeployment() error {
	manifest, err := getVersionManifest()
	if err != nil {
		return fmt.Errorf("failed to get version manifest: %v", err)
	}

	archiveFiles, err := getDownloadableArchives(manifest)
	if err != nil {
		return fmt.Errorf("failed to get downloadable archives: %v", err)
	}

	clientInfo, err := getClientInfo()
	if err != nil {
		return fmt.Errorf("failed to get client info: %v", err)
	}

	baseURL := fmt.Sprintf("https://setup.rbxcdn.com/%s-%%s", clientInfo.ClientVersionUpload)

	configDir := getConfigDirectory()

	if err := os.MkdirAll(path.Join(configDir, "versions"), os.ModePerm); err != nil {
		return fmt.Errorf("failed to create versions directory: %v", err)
	}

	versionsDir := path.Join(configDir, "versions")

	for _, archiveFile := range archiveFiles {
		archiveURL := fmt.Sprintf(baseURL, archiveFile)

		response, err := http.Get(archiveURL)
		if err != nil {
			return fmt.Errorf("failed to get archive %s: %v", archiveFile, err)
		}
		defer response.Body.Close()

		bodyBytes, err := io.ReadAll(response.Body)
		if err != nil {
			return fmt.Errorf("failed to read archive body: %v", err)
		}

		zipReader, err := zip.NewReader(bytes.NewReader(bodyBytes), int64(len(bodyBytes)))
		if err != nil {
			return fmt.Errorf("failed to create zip reader for %s: %v", archiveFile, err)
		}

		log.Printf("Extracting contents of %s to %s (relative)", archiveFile, path.Join(EXTRACTION_ROOTS[archiveFile]))

		for _, zipFile := range zipReader.File {
			extractionRoot := EXTRACTION_ROOTS[archiveFile]
			filePath := filepath.Join(versionsDir, clientInfo.ClientVersionUpload, extractionRoot, zipFile.Name)

			if zipFile.FileInfo().IsDir() {
				if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
					return fmt.Errorf("failed to create directory %s: %v", filePath, err)
				}
				continue
			}

			if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
				return fmt.Errorf("failed to create directory %s: %v", filepath.Dir(filePath), err)
			}

			outFile, err := os.Create(filePath)
			if err != nil {
				return fmt.Errorf("failed to create file %s: %v", filePath, err)
			}
			defer outFile.Close()

			fileContent, err := zipFile.Open()
			if err != nil {
				return fmt.Errorf("failed to open file %s in zip %s: %v", zipFile.Name, archiveFile, err)
			}
			defer fileContent.Close()

			if _, err := io.Copy(outFile, fileContent); err != nil {
				return fmt.Errorf("failed to copy file %s from zip %s: %v", zipFile.Name, archiveFile, err)
			}
		}
	}

	if err := os.WriteFile(path.Join(versionsDir, clientInfo.ClientVersionUpload, "AppSettings.xml"), []byte(appsettings), os.ModePerm); err != nil {
		return fmt.Errorf("failed to write appsettings: %s", err)
	}

	return nil
}

func getLatestExecutablePath() (string, error) {
	latestClientInfo, err := getClientInfo()
	if err != nil {
		return "", fmt.Errorf("failed to get client info: %v", err)
	}

	confgDir := getConfigDirectory()
	versionsDir := path.Join(confgDir, "versions")

	clientVersion := latestClientInfo.ClientVersionUpload
	executablePath := path.Join(versionsDir, clientVersion, "RobloxPlayerBeta.exe")

	if _, err := os.Stat(executablePath); os.IsNotExist(err) {
		saveError := saveDeployment()

		if saveError != nil {
			return "", fmt.Errorf("%v", saveError)
		}
	}

	return executablePath, nil
}

func safeLaunch(executable string, args ...string) error {
	cmd := exec.Command(executable, args...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to launch %s: %v", executable, err)
	}
	if cmd.Process != nil {
		cmd.Process.Release()
	}
	return nil
}

func launchRoblox(playerArgs ...string) error {
	executablePath, err := getLatestExecutablePath()

	if err != nil {
		return fmt.Errorf("%v", err)
	}

	if err := safeLaunch(executablePath, playerArgs...); err != nil {
		return fmt.Errorf("%s", err)
	}

	return nil
}
