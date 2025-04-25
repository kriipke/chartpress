package server

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
        "gopkg.in/yaml.v2"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
        "helm.sh/helm/v3/pkg/chartutil"
)

// Config holds the configuration details for the chart generation
type Config struct {
	UmbrellaChartName string     `json:"umbrellaChartName"` // Name of the umbrella chart
	Subcharts         []Subchart `json:"subcharts"`         // List of subcharts
}

// Subchart represents a single subchart's metadata
type Subchart struct {
	Name     string `json:"name"`     // Name of the subchart
	Workload string `json:"workload"` // Type of workload: deployment, statefulset, or daemonset
}

// Start initializes and starts the HTTP server
func Start() {
	// Define the /generate endpoint handler
	http.HandleFunc("/generate", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Handling /generate endpoint\n")

		// Allow all origins for CORS
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Handle preflight OPTIONS request for CORS
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}

		// Handle the actual POST request
		handleGenerate(w, r)
	})

	// Retrieve the port number from environment variables
	port := getPort()

	// Log and start the server
	fmt.Printf("Starting server on port %s...\n", port)
	log.Printf("Server is starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil)) // Fatal will log and exit on error
}

// getPort retrieves the server port from the environment or defaults to 8080
func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port
		log.Println("Environment variable PORT not set, defaulting to 8080")
	}
	return port
}

// handleGenerate processes the /generate POST request
func handleGenerate(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request to /generate endpoint")

	// Ensure only POST requests are allowed
	if r.Method != http.MethodPost {
		log.Printf("Invalid method: %s. Only POST is allowed.", r.Method)
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	// Decode the JSON payload into the Config struct
	var cfg Config
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		log.Printf("Failed to parse JSON: %v", err)
		http.Error(w, fmt.Sprintf("Failed to parse JSON: %v", err), http.StatusBadRequest)
		return
	}
	log.Printf("Loaded config: %+v", cfg)

	// Validate the configuration
	if err := validateConfig(cfg); err != nil {
		log.Printf("Invalid config: %v", err)
		http.Error(w, fmt.Sprintf("Invalid config: %v", err), http.StatusBadRequest)
		return
	}

	// Save the configuration to a file
	configFilePath := "./chartpress.yaml"
	log.Printf("Saving configuration to %s", configFilePath)
	configFile, err := os.Create(configFilePath)
	if err != nil {
		log.Printf("Failed to create config file: %v", err)
		http.Error(w, fmt.Sprintf("Failed to create config file: %v", err), http.StatusInternalServerError)
		return
	}
	// Ensure the file is properly closed
	defer func() {
		if err := configFile.Close(); err != nil {
			log.Printf("Error closing config file: %v", err)
		}
	}()

	// Write the configuration to the file in JSON format
	if err := json.NewEncoder(configFile).Encode(cfg); err != nil {
		log.Printf("Failed to write config file: %v", err)
		http.Error(w, fmt.Sprintf("Failed to write config file: %v", err), http.StatusInternalServerError)
		return
	}

	// Generate the chart from the configuration
	outputDir, err := generateChart(cfg)
	if err != nil {
		log.Printf("Failed to generate chart: %v", err)
		http.Error(w, fmt.Sprintf("Failed to generate chart: %v", err), http.StatusInternalServerError)
		return
	}
	log.Printf("Generated chart at %s", outputDir)

	// Create a zip file of the generated chart
	zipFilePath := fmt.Sprintf("%s.zip", outputDir)
	if err := zipOutputDir(outputDir, zipFilePath); err != nil {
		log.Printf("Failed to create zip file: %v", err)
		http.Error(w, fmt.Sprintf("Failed to create zip file: %v", err), http.StatusInternalServerError)
		return
	}

	// Serve the zip file to the client
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(zipFilePath)))
	w.Header().Set("Content-Type", "application/zip")
	http.ServeFile(w, r, zipFilePath)
}


// generateChart generates the Helm chart based on the configuration
func generateChart(cfg Config) (string, error) {
	
    umbrellaChartPath := "templates/umbrella"
    subchartChartPath := "templates/subchart"

    subcharts := cfg.Subcharts

    // Load the umbrella chart
    ch, err := loadChart(umbrellaChartPath)
    if err != nil {
        fmt.Println("Error loading umbrella chart:", err)
        return "error", err
    }

    // Rename the umbrella chart
    chartName := cfg.UmbrellaChartName
    chNew, err := renameChart(ch, chartName )
    if err != nil {
        fmt.Println("Error renaming umbrella chart:", err)
        return "error", err
    }

    // Add each subchart from config
    for _, sc := range subcharts {
        chNew, err = newSubchart(chNew, subchartChartPath, sc.Name)
        if err != nil {
            fmt.Printf("Error adding subchart '%s': %v\n", sc.Name, err)
            return sc.Name, err
        }
    }

    // Define the output directory
    outputDir := filepath.Join("output", chartName)

    // Save the chart to the output directory
    if err := chartutil.SaveDir(chNew, outputDir); err != nil {
        return "", fmt.Errorf("error saving chart to directory: %w", err)
    }

    return outputDir, nil
}
// validateConfig validates the provided configuration
func validateConfig(cfg Config) error {
	// Example validation: Ensure the umbrella chart name is not empty
	if strings.TrimSpace(cfg.UmbrellaChartName) == "" {
		return fmt.Errorf("umbrellaChartName cannot be empty")
	}

	// Ensure each subchart has a name and a valid workload type
	for _, subchart := range cfg.Subcharts {
		if strings.TrimSpace(subchart.Name) == "" {
			return fmt.Errorf("subchart name cannot be empty")
		}
		if subchart.Workload != "deployment" && subchart.Workload != "statefulset" && subchart.Workload != "daemonset" {
			return fmt.Errorf("invalid workload type for subchart %s: %s", subchart.Name, subchart.Workload)
		}
	}
	return nil
}


func loadChart(chartPath string) (*chart.Chart, error) {
    ch, err := loader.Load(chartPath)
    if err != nil {
        return nil, fmt.Errorf("failed to load chart from %s: %w", chartPath, err)
    }
    return ch, nil
}

func renameChart(ch *chart.Chart, newName string) (*chart.Chart, error) {
    originalName := ch.Metadata.Name
    ch.Metadata.Name = newName

    // Update references in templates
    for _, tmpl := range ch.Templates {
        tmpl.Data = []byte(strings.ReplaceAll(string(tmpl.Data), originalName, newName))
    }

    // Update references in values.yaml
    if ch.Values != nil {
        valuesYAML, err := yaml.Marshal(ch.Values)
        if err != nil {
            return nil, fmt.Errorf("failed to marshal values.yaml: %w", err)
        }
        updatedValues := strings.ReplaceAll(string(valuesYAML), originalName, newName)
        var newValues map[string]interface{}
        if err := yaml.Unmarshal([]byte(updatedValues), &newValues); err != nil {
            return nil, fmt.Errorf("failed to unmarshal updated values.yaml: %w", err)
        }
        ch.Values = newValues
    }

    // Update references in additional files
    for _, file := range ch.Files {
        file.Data = []byte(strings.ReplaceAll(string(file.Data), originalName, newName))
    }

    return ch, nil
}



// with the parent chart's name in templates and files, and adds it as a dependency.
func newSubchart(parentChart *chart.Chart, subchartPath, subchartName string) (*chart.Chart, error) {
    // Load the subchart
    subchart, err := loader.Load(subchartPath)
    if err != nil {
        return nil, fmt.Errorf("failed to load subchart: %w", err)
    }

    // Rename the subchart
    subchart.Metadata.Name = subchartName

    // Define placeholders and their replacements
    replacements := map[string]string{
        "umbrella-chart": parentChart.Metadata.Name,
        "component":      subchartName,
    }

    // Replace placeholders in templates
    for _, tmpl := range subchart.Templates {
        content := string(tmpl.Data)
        for old, new := range replacements {
            content = strings.ReplaceAll(content, old, new)
        }
        tmpl.Data = []byte(content)
    }

    // Replace placeholders in additional files
    for _, file := range subchart.Files {
        content := string(file.Data)
        for old, new := range replacements {
            content = strings.ReplaceAll(content, old, new)
        }
        file.Data = []byte(content)
    }


    // Add the subchart to the parent chart's dependencies
    parentChart.AddDependency(subchart)

    // Update the parent chart's metadata dependencies
    // Assuming parentChart.Metadata.Dependencies is of type []*chart.Dependency
    newDependency := &chart.Dependency{
        Name:       subchartName,
        Version:    subchart.Metadata.Version,
        Repository: fmt.Sprintf("file://charts/%s", subchartName),
    }
    parentChart.Metadata.Dependencies = append(parentChart.Metadata.Dependencies, newDependency)
    return parentChart, nil
}

// zipOutputDir creates a zip archive of the output directory
func zipOutputDir(outputDir, zipFilePath string) error {
	zipFile, err := os.Create(zipFilePath)
	if err != nil {
		return fmt.Errorf("failed to create zip file: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	err = filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing path %s: %w", path, err)
		}
		// Skip directories as they will be included implicitly
		if info.IsDir() {
			return nil
		}

		// Create a zip entry for the file
		relPath, err := filepath.Rel(outputDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}
		zipEntry, err := zipWriter.Create(relPath)
		if err != nil {
			return fmt.Errorf("failed to create zip entry for %s: %w", relPath, err)
		}

		// Write the file contents to the zip entry
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", path, err)
		}
		defer file.Close()

		if _, err := io.Copy(zipEntry, file); err != nil {
			return fmt.Errorf("failed to write file %s to zip: %w", path, err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("error creating zip archive: %w", err)
	}
	return nil
}
