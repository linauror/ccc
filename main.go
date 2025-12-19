package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// Configuration represents a Claude Code configuration entry
type Configuration struct {
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
	Active  bool   `json:"active"`
}

// ConfigFile represents the structure of the configuration file
type ConfigFile struct {
	Configurations []Configuration `json:"configurations"`
}

var configFileName = "ccc-config.json"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "list", "ls":
		if err := listConfigurations(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "add":
		addCmd := flag.NewFlagSet("add", flag.ExitOnError)
		name := addCmd.String("n", "", "Configuration name (required)")
		baseURL := addCmd.String("u", "", "Base URL for API (required)")
		apiKey := addCmd.String("k", "", "API key (required)")

		addCmd.Parse(os.Args[2:])

		if *name == "" || *baseURL == "" || *apiKey == "" {
			fmt.Fprintln(os.Stderr, "Error: name, base-url, and api-key are required")
			addCmd.PrintDefaults()
			os.Exit(1)
		}

		if err := addConfiguration(*name, *baseURL, *apiKey); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "update":
		updateCmd := flag.NewFlagSet("update", flag.ExitOnError)
		name := updateCmd.String("n", "", "Configuration name (required)")
		baseURL := updateCmd.String("u", "", "New base URL for API")
		apiKey := updateCmd.String("k", "", "New API key")

		updateCmd.Parse(os.Args[2:])

		if *name == "" {
			fmt.Fprintln(os.Stderr, "Error: name is required")
			updateCmd.PrintDefaults()
			os.Exit(1)
		}

		if err := updateConfiguration(*name, *baseURL, *apiKey); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "delete":
		deleteCmd := flag.NewFlagSet("delete", flag.ExitOnError)
		name := deleteCmd.String("n", "", "Configuration name (required)")

		deleteCmd.Parse(os.Args[2:])

		if *name == "" {
			fmt.Fprintln(os.Stderr, "Error: name is required")
			deleteCmd.PrintDefaults()
			os.Exit(1)
		}

		if err := deleteConfiguration(*name); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "activate":
		activateCmd := flag.NewFlagSet("activate", flag.ExitOnError)
		name := activateCmd.String("n", "", "Configuration name (required)")

		activateCmd.Parse(os.Args[2:])

		if *name == "" {
			fmt.Fprintln(os.Stderr, "Error: name is required")
			activateCmd.PrintDefaults()
			os.Exit(1)
		}

		if err := activateConfiguration(*name); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

// printUsage prints the usage information
func printUsage() {
	fmt.Println("CCC - Claude Code Configuration Manager")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  ccc <command> [flags]")
	fmt.Println()
	fmt.Println("Available commands:")
	fmt.Println("  list      List all configurations")
	fmt.Println("  ls        Alias for list")
	fmt.Println("  add       Add a new configuration")
	fmt.Println("  update    Update an existing configuration")
	fmt.Println("  delete    Delete a configuration")
	fmt.Println("  activate  Activate a configuration and apply settings")
	fmt.Println()
	fmt.Println("Use 'ccc <command> -h' for more information about a command.")
}

// getConfigPath returns the path to the configuration file
func getConfigPath() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}

	execDir := filepath.Dir(execPath)
	configPath := filepath.Join(execDir, configFileName)

	return configPath, nil
}

// loadConfig loads the configuration from file
func loadConfig() (*ConfigFile, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return nil, err
	}

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// File doesn't exist, try to auto-import
		var importedConfig *Configuration

		switch runtime.GOOS {
		case "windows":
			importedConfig = importFromWindows()
		case "linux", "darwin":
			importedConfig = importFromUnixSettings()
		}

		if importedConfig != nil {
			// Create config with imported configuration
			config := &ConfigFile{
				Configurations: []Configuration{*importedConfig},
			}

			// Save the imported configuration
			if err := saveConfig(config); err == nil {
				fmt.Printf("Auto-imported configuration '%s' from existing settings\n", importedConfig.Name)
			}

			return config, nil
		}

		// Return empty config if no import found
		return &ConfigFile{
			Configurations: []Configuration{},
		}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var configFile ConfigFile
	if err := json.Unmarshal(data, &configFile); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &configFile, nil
}

// saveConfig saves the configuration to file
func saveConfig(config *ConfigFile) error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// findConfiguration finds a configuration by name
func findConfiguration(config *ConfigFile, name string) *Configuration {
	for i := range config.Configurations {
		if config.Configurations[i].Name == name {
			return &config.Configurations[i]
		}
	}
	return nil
}

// setActiveConfiguration sets a configuration as active and deactivates others
func setActiveConfiguration(config *ConfigFile, activeName string) {
	for i := range config.Configurations {
		config.Configurations[i].Active = config.Configurations[i].Name == activeName
	}
}

// listConfigurations lists all configurations
func listConfigurations() error {
	config, err := loadConfig()
	if err != nil {
		return err
	}

	if len(config.Configurations) == 0 {
		fmt.Println("No configurations found.")
		return nil
	}

	// Sort configurations by name for consistent display
	sort.Slice(config.Configurations, func(i, j int) bool {
		return config.Configurations[i].Name < config.Configurations[j].Name
	})

	// Calculate column widths
	nameWidth := 4   // "Name" length
	statusWidth := 6 // "Status" length
	urlWidth := 8    // "Base URL" length
	apiKeyWidth := 7 // "API Key" length

	for _, conf := range config.Configurations {
		if len(conf.Name) > nameWidth {
			nameWidth = len(conf.Name)
		}

		status := "Inactive"
		if conf.Active {
			status = "Active"
		}
		if len(status) > statusWidth {
			statusWidth = len(status)
		}

		if len(conf.BaseURL) > urlWidth {
			urlWidth = len(conf.BaseURL)
		}

		maskedKey := maskAPIKey(conf.APIKey)
		if len(maskedKey) > apiKeyWidth {
			apiKeyWidth = len(maskedKey)
		}
	}

	// Add padding
	nameWidth += 2
	statusWidth += 2
	urlWidth += 2
	apiKeyWidth += 2

	// Print table header
	fmt.Printf("%-*s%-*s%-*s%-*s\n",
		nameWidth, "Name",
		statusWidth, "Status",
		urlWidth, "Base URL",
		apiKeyWidth, "API Key")
	fmt.Printf("%-*s%-*s%-*s%-*s\n",
		nameWidth, strings.Repeat("-", nameWidth-2),
		statusWidth, strings.Repeat("-", statusWidth-2),
		urlWidth, strings.Repeat("-", urlWidth-2),
		apiKeyWidth, strings.Repeat("-", apiKeyWidth-2))

	// Print table rows
	for _, conf := range config.Configurations {
		status := "Inactive"
		if conf.Active {
			status = "Active"
		}

		fmt.Printf("%-*s%-*s%-*s%-*s\n",
			nameWidth, conf.Name,
			statusWidth, status,
			urlWidth, conf.BaseURL,
			apiKeyWidth, maskAPIKey(conf.APIKey))
	}

	return nil
}

// addConfiguration adds a new configuration
func addConfiguration(name, baseURL, apiKey string) error {
	config, err := loadConfig()
	if err != nil {
		return err
	}

	// Check if configuration with the same name already exists
	if findConfiguration(config, name) != nil {
		return fmt.Errorf("configuration with name '%s' already exists", name)
	}

	// Create new configuration
	newConf := Configuration{
		Name:    name,
		BaseURL: baseURL,
		APIKey:  apiKey,
		Active:  false,
	}

	// If this is the first configuration, make it active
	if len(config.Configurations) == 0 {
		newConf.Active = true
	}

	config.Configurations = append(config.Configurations, newConf)

	if err := saveConfig(config); err != nil {
		return err
	}

	if newConf.Active {
		fmt.Printf("Configuration '%s' added and activated successfully.\n", name)
	} else {
		fmt.Printf("Configuration '%s' added successfully.\n", name)
	}

	return nil
}

// updateConfiguration updates an existing configuration
func updateConfiguration(name, baseURL, apiKey string) error {
	config, err := loadConfig()
	if err != nil {
		return err
	}

	conf := findConfiguration(config, name)
	if conf == nil {
		return fmt.Errorf("configuration with name '%s' not found", name)
	}

	// Update fields if provided
	if baseURL != "" {
		conf.BaseURL = baseURL
	}
	if apiKey != "" {
		conf.APIKey = apiKey
	}

	if err := saveConfig(config); err != nil {
		return err
	}

	fmt.Printf("Configuration '%s' updated successfully.\n", name)
	return nil
}

// deleteConfiguration deletes a configuration
func deleteConfiguration(name string) error {
	config, err := loadConfig()
	if err != nil {
		return err
	}

	conf := findConfiguration(config, name)
	if conf == nil {
		return fmt.Errorf("configuration with name '%s' not found", name)
	}

	// Prevent deletion of active configuration
	if conf.Active {
		return fmt.Errorf("cannot delete active configuration '%s'. Please activate another configuration first", name)
	}

	// Remove the configuration
	var newConfigurations []Configuration
	for _, c := range config.Configurations {
		if c.Name != name {
			newConfigurations = append(newConfigurations, c)
		}
	}

	config.Configurations = newConfigurations

	if err := saveConfig(config); err != nil {
		return err
	}

	fmt.Printf("Configuration '%s' deleted successfully.\n", name)
	return nil
}

// activateConfiguration activates a configuration and applies settings
func activateConfiguration(name string) error {
	config, err := loadConfig()
	if err != nil {
		return err
	}

	conf := findConfiguration(config, name)
	if conf == nil {
		return fmt.Errorf("configuration with name '%s' not found", name)
	}

	// Set this configuration as active
	setActiveConfiguration(config, name)

	if err := saveConfig(config); err != nil {
		return err
	}

	fmt.Printf("Configuration '%s' activated successfully.\n", name)

	// Apply the configuration settings
	switch runtime.GOOS {
	case "windows":
		return setWindowsEnvironmentVariables(conf)
	case "linux", "darwin":
		return setUnixSettingsFile(conf)
	default:
		fmt.Fprintf(os.Stderr, "Warning: Unsupported platform (%s), settings not applied\n", runtime.GOOS)
	}

	return nil
}

// ClaudeSettings represents the structure of Claude's settings.json file
type ClaudeSettings struct {
	Env map[string]interface{} `json:"env"`
}

// NewClaudeSettings creates a new settings structure with default values
func NewClaudeSettings() ClaudeSettings {
	return ClaudeSettings{
		Env: map[string]interface{}{
			"ANTHROPIC_AUTH_TOKEN":                "",
			"ANTHROPIC_BASE_URL":                   "",
			"API_TIMEOUT_MS":                       "3000000",
			"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": 1,
		},
	}
}


// setWindowsEnvironmentVariables sets environment variables on Windows
func setWindowsEnvironmentVariables(activeConfig *Configuration) error {
	// Set environment variables for the current process
	if err := os.Setenv("ANTHROPIC_BASE_URL", activeConfig.BaseURL); err != nil {
		return fmt.Errorf("failed to set ANTHROPIC_BASE_URL: %w", err)
	}

	if err := os.Setenv("ANTHROPIC_AUTH_TOKEN", activeConfig.APIKey); err != nil {
		return fmt.Errorf("failed to set ANTHROPIC_AUTH_TOKEN: %w", err)
	}

	fmt.Printf("Environment variables set for active configuration '%s':\n", activeConfig.Name)
	fmt.Printf("ANTHROPIC_BASE_URL=%s\n", activeConfig.BaseURL)
	fmt.Printf("ANTHROPIC_AUTH_TOKEN=%s\n", maskAPIKey(activeConfig.APIKey))

	// Also set them permanently using setx
	fmt.Println("\nSetting permanent environment variables...")

	setxBaseURL := exec.Command("setx", "ANTHROPIC_BASE_URL", activeConfig.BaseURL)
	if output, err := setxBaseURL.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to set permanent ANTHROPIC_BASE_URL: %v\n", err)
		fmt.Fprintf(os.Stderr, "Output: %s\n", string(output))
	} else {
		fmt.Println("Successfully set permanent ANTHROPIC_BASE_URL")
	}

	setxAuthToken := exec.Command("setx", "ANTHROPIC_AUTH_TOKEN", activeConfig.APIKey)
	if output, err := setxAuthToken.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to set permanent ANTHROPIC_AUTH_TOKEN: %v\n", err)
		fmt.Fprintf(os.Stderr, "Output: %s\n", string(output))
	} else {
		fmt.Println("Successfully set permanent ANTHROPIC_AUTH_TOKEN")
	}

	fmt.Println("\nNote: Permanent environment variables will be available in new command prompt windows.")

	return nil
}

// setUnixSettingsFile updates the Claude settings.json file on Linux/macOS
func setUnixSettingsFile(activeConfig *Configuration) error {
	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Create .claude directory if it doesn't exist
	claudeDir := filepath.Join(homeDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return fmt.Errorf("failed to create .claude directory: %w", err)
	}

	// Path to settings.json
	settingsPath := filepath.Join(claudeDir, "settings.json")

	// Read existing settings if file exists
	var settings ClaudeSettings
	if data, err := os.ReadFile(settingsPath); err == nil {
		if err := json.Unmarshal(data, &settings); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to parse existing settings.json, creating new one: %v\n", err)
			settings = NewClaudeSettings()
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to read settings.json: %w", err)
	} else {
		// File doesn't exist, create new settings with defaults
		settings = NewClaudeSettings()
	}

	// Ensure Env map exists
	if settings.Env == nil {
		settings.Env = make(map[string]interface{})
	}

	// Update settings with active configuration
	settings.Env["ANTHROPIC_AUTH_TOKEN"] = activeConfig.APIKey
	settings.Env["ANTHROPIC_BASE_URL"] = activeConfig.BaseURL

	// Ensure default values are present
	if _, exists := settings.Env["API_TIMEOUT_MS"]; !exists {
		settings.Env["API_TIMEOUT_MS"] = "3000000"
	}
	if _, exists := settings.Env["CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC"]; !exists {
		settings.Env["CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC"] = 1
	}

	// Write updated settings back to file
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write settings.json: %w", err)
	}

	fmt.Printf("Claude settings updated for active configuration '%s':\n", activeConfig.Name)
	fmt.Printf("Settings file: %s\n", settingsPath)
	fmt.Printf("Base URL: %s\n", activeConfig.BaseURL)
	fmt.Printf("API Key: %s\n", maskAPIKey(activeConfig.APIKey))

	return nil
}

// extractDomainFromURL extracts the middle part of the domain for configuration name
func extractDomainFromURL(baseURL string) string {
	if baseURL == "" {
		return "default"
	}

	// Parse URL
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return "default"
	}

	// Get hostname
	hostname := parsedURL.Hostname()
	if hostname == "" {
		return "default"
	}

	// Remove www. prefix
	hostname = strings.TrimPrefix(hostname, "www.")

	// Split by dots
	parts := strings.Split(hostname, ".")

	// Return middle part if available
	if len(parts) >= 2 {
		// For domain like api.anthropic.com, return "anthropic"
		if len(parts) >= 3 {
			return parts[1]
		}
		// For domain like anthropic.com, return "anthropic"
		return parts[0]
	}

	// Fallback to first part
	return parts[0]
}

// importFromWindows imports configuration from Windows environment variables
func importFromWindows() *Configuration {
	baseURL := os.Getenv("ANTHROPIC_BASE_URL")
	apiKey := os.Getenv("ANTHROPIC_AUTH_TOKEN")

	if baseURL == "" || apiKey == "" {
		return nil
	}

	name := extractDomainFromURL(baseURL)
	return &Configuration{
		Name:    name,
		BaseURL: baseURL,
		APIKey:  apiKey,
		Active:  true,
	}
}

// importFromUnixSettings imports configuration from Unix settings.json file
func importFromUnixSettings() *Configuration {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	settingsPath := filepath.Join(homeDir, ".claude", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil
	}

	var settings ClaudeSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil
	}

	if settings.Env == nil {
		return nil
	}

	baseURL, baseURLExists := settings.Env["ANTHROPIC_BASE_URL"].(string)
	apiKey, apiKeyExists := settings.Env["ANTHROPIC_AUTH_TOKEN"].(string)

	if !baseURLExists || !apiKeyExists || baseURL == "" || apiKey == "" {
		return nil
	}

	name := extractDomainFromURL(baseURL)
	return &Configuration{
		Name:    name,
		BaseURL: baseURL,
		APIKey:  apiKey,
		Active:  true,
	}
}

// maskAPIKey masks the API key for display
func maskAPIKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return strings.Repeat("*", len(apiKey))
	}
	return apiKey[:4] + strings.Repeat("*", len(apiKey)-8) + apiKey[len(apiKey)-4:]
}
