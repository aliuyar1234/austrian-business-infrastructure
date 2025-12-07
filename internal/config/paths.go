package config

import (
	"os"
	"path/filepath"
	"runtime"
)

const (
	// AppName is the application name used for config directories
	AppName = "fo"
	// CredentialFileName is the name of the encrypted credentials file
	CredentialFileName = "accounts.enc"
)

// GetConfigDir returns the platform-appropriate config directory.
// If override is non-empty, it is used instead of the default.
// Creates the directory if it doesn't exist.
func GetConfigDir(override string) (string, error) {
	if override != "" {
		if err := os.MkdirAll(override, 0700); err != nil {
			return "", err
		}
		return override, nil
	}

	dir := getDefaultConfigDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return dir, nil
}

// getDefaultConfigDir returns the platform-specific default config directory
func getDefaultConfigDir() string {
	switch runtime.GOOS {
	case "windows":
		// Use %APPDATA%\fo on Windows
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
		}
		return filepath.Join(appData, AppName)
	default:
		// Use XDG_CONFIG_HOME/fo on Linux and macOS
		xdgConfig := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfig == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				home = "."
			}
			xdgConfig = filepath.Join(home, ".config")
		}
		return filepath.Join(xdgConfig, AppName)
	}
}

// GetCredentialPath returns the full path to the credential file
func GetCredentialPath(configDir string) string {
	return filepath.Join(configDir, CredentialFileName)
}
