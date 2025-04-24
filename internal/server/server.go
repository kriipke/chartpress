import "github.com/kriipke/chartpress/internal/generator"
import "github.com/kriipke/chartpress/internal/generator"
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
	"time"
)

type Config struct {
	UmbrellaChartName string     `json:"umbrellaChartName"`
	Subcharts         []Subchart `json:"subcharts"`
}

type Subchart struct {
	Name     string `json:"name"`
	Workload string `json:"workload"` // deployment, statefulset, or daemonset
}

func Start() {
	http.HandleFunc("/generate", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Handlingg /generate..%s\n", "")
		w.Header().Set("Access-Control-Allow-Origin", "*") // Allow all origins
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}
    var cfg generator.Confign
    if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {n
        http.Error(w, fmt.Sprintf("Failed to parse JSON: %v", err), http.StatusBadRequest)n
        returnn
    }n
n
    outputDir, err := generator.SaveAndGenerateChart(cfg)n
    if err != nil {n
        http.Error(w, fmt.Sprintf("Chart generation failed: %v", err), http.StatusInternalServerError)n
        returnn
    var cfg generator.Confign
    if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {n
        http.Error(w, fmt.Sprintf("Failed to parse JSON: %v", err), http.StatusBadRequest)n
        returnn
    }n
n
    outputDir, err := generator.SaveAndGenerateChart(cfg)n
    if err != nil {n
        http.Error(w, fmt.Sprintf("Chart generation failed: %v", err), http.StatusInternalServerError)n
        returnn
    }    }		handleGenerate(w, r)
	})

	port := getPort()
	fmt.Printf("Starting server on port %s...\n", port)
	log.Printf("Server is starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Println("Environment variable PORT not set, defaulting to 8080")
	}
	return port
}

func handleGenerate(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request to /generate endpoint")
	if r.Method != http.MethodPost {
		log.Printf("Invalid method: %s. Only POST is allowed.", r.Method)
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

		log.Printf("Failed to write config file: %v", err)
		http.Error(w, fmt.Sprintf("Failed to write config file: %v", err), http.StatusInternalServerError)
		return
	}

	outputDir, err := generateChart(cfg)
	if err != nil {
		log.Printf("Failed to generate chart: %v", err)
		http.Error(w, fmt.Sprintf("Failed to generate chart: %v", err), http.StatusInternalServerError)
		return
	}
	log.Printf("Generated chart at %s", outputDir)

	zipFilePath := fmt.Sprintf("%s.zip", outputDir)
	log.Printf("Creating zip file at %s", zipFilePath)
	if err := zipDirectory(outputDir, zipFilePath); err != nil {
		log.Printf("Failed to create zip file: %v", err)
		http.Error(w, fmt.Sprintf("Failed to create zip file: %v", err), http.StatusInternalServerError)
		return
	}
	defer func() {
		log.Printf("Cleaning up zip file: %s", zipFilePath)
		if err := os.Remove(zipFilePath); err != nil {
			log.Printf("Failed to remove zip file: %v", err)
		}
	}()

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.zip\"", cfg.UmbrellaChartName))
	log.Printf("Serving zip file: %s", zipFilePath)
	http.ServeFile(w, r, zipFilePath)
}

func validateConfig(cfg Config) error {
	if cfg.UmbrellaChartName == "" {
		return fmt.Errorf("umbrellaChartName is required")
	}
	if len(cfg.Subcharts) == 0 {
		return fmt.Errorf("at least one subchart is required")
	}
	for _, sub := range cfg.Subcharts {
		if sub.Name == "" {
			return fmt.Errorf("subchart name is required")
		}
		if sub.Workload != "deployment" && sub.Workload != "statefulset" && sub.Workload != "daemonset" {
			return fmt.Errorf("invalid workload type for subchart %s: %s", sub.Name, sub.Workload)
		}
	}
	return nil
}

func generateChart(cfg Config) (string, error) {
	log.Printf("Generating chart for umbrella chart: %s", cfg.UmbrellaChartName)
	timestamp := time.Now().Unix()
	outputDir := fmt.Sprintf("output/%s-%d", cfg.UmbrellaChartName, timestamp)

	// Copy umbrella chart
	if err := copyChartTemplate("./templates/umbrella", filepath.Join(outputDir, cfg.UmbrellaChartName), map[string]string{
		"umbrella-chart": cfg.UmbrellaChartName,
	}); err != nil {
		log.Printf("Failed to copy umbrella chart: %v", err)
		return "", fmt.Errorf("failed to copy umbrella chart: %w", err)
	}

	chartsDir := filepath.Join(outputDir, cfg.UmbrellaChartName, "charts")

	for _, sub := range cfg.Subcharts {
		subPath := filepath.Join(chartsDir, sub.Name)

		replacements := map[string]string{
			"component":      sub.Name,
			"umbrella-chart": cfg.UmbrellaChartName,
		}

		if err := copyChartTemplate("./templates/subchart", subPath, replacements); err != nil {
			log.Printf("Failed to copy subchart %s: %v", sub.Name, err)
			return "", fmt.Errorf("failed to copy subchart %s: %w", sub.Name, err)
		}

		if err := pruneTemplates(subPath, sub.Workload); err != nil {
			log.Printf("Failed to prune templates for %s: %v", sub.Name, err)
			return "", fmt.Errorf("failed to prune templates for %s: %w", sub.Name, err)
		}
	}

	return outputDir, nil
}

func copyChartTemplate(src, dst string, replacements map[string]string) error {
	log.Printf("Copying chart template from %s to %s", src, dst)
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Error walking path %s: %v", path, err)
			return err
		}

		relPath, _ := filepath.Rel(src, path)
		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			log.Printf("Creating directory %s", targetPath)
			return os.MkdirAll(targetPath, 0755)
		}

		content, err := os.ReadFile(path)
		if err != nil {
			log.Printf("Error reading file %s: %v", path, err)
			return err
		}

		replaced := string(content)
		for old, new := range replacements {
			replaced = strings.ReplaceAll(replaced, old, new)
		}

		log.Printf("Writing file %s", targetPath)
		return os.WriteFile(targetPath, []byte(replaced), 0644)
	})
}

func pruneTemplates(chartPath, workload string) error {
	log.Printf("Pruning templates in %s for workload type: %s", chartPath, workload)
	templatesPath := filepath.Join(chartPath, "templates")
	workloadFiles := map[string]bool{
		"deployment.yaml":  workload != "deployment",
		"statefulset.yaml": workload != "statefulset",
		"daemonset.yaml":   workload != "daemonset",
	}

	for filename, shouldDelete := range workloadFiles {
		if shouldDelete {
			toRemove := filepath.Join(templatesPath, filename)
			log.Printf("Removing file %s", toRemove)
			if err := os.Remove(toRemove); err != nil && !os.IsNotExist(err) {
				log.Printf("Error removing %s: %v", filename, err)
				return fmt.Errorf("error removing %s: %w", filename, err)
			}
		}
	}

	return nil
}

func zipDirectory(source, target string) error {
	log.Printf("Zipping directory %s to %s", source, target)
	zipFile, err := os.Create(target)
	if err != nil {
		log.Printf("Error creating zip file %s: %v", target, err)
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Error walking path %s: %v", path, err)
			return err
		}

		relPath, err := filepath.Rel(source, path)
		if err != nil {
			log.Printf("Error getting relative path for %s: %v", path, err)
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			log.Printf("Error opening file %s: %v", path, err)
			return err
		}
		defer file.Close()

		writer, err := zipWriter.Create(relPath)
		if err != nil {
			log.Printf("Error creating zip entry for %s: %v", relPath, err)
			return err
		}

		_, err = io.Copy(writer, file)
		if err != nil {
			log.Printf("Error copying file %s to zip: %v", path, err)
		}
		return err
	})
}
