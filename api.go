package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"github.com/kriipke/chartpress/cmd"

)

type Config struct {
	UmbrellaChartName string     `json:"umbrellaChartName"`
	Subcharts         []Subchart `json:"subcharts"`
}

type Subchart struct {
	Name     string `json:"name"`
	Workload string `json:"workload"` // deployment, statefulset, or daemonset
}

func main() {
	http.HandleFunc("/generate", handleGenerate)

	port := "8080"
	fmt.Printf("Starting server on port %s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var cfg Config
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Save the configuration as chartpress.yaml
	configFilePath := "./chartpress.yaml"
	configFile, err := os.Create(configFilePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create config file: %v", err), http.StatusInternalServerError)
		return
	}
	defer func() {
		if err := configFile.Close(); err != nil {
			log.Printf("Error closing config file: %v", err)
		}
	}()

	if err := json.NewEncoder(configFile).Encode(cfg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to write config file: %v", err), http.StatusInternalServerError)
		return
	}

	outputDir, err := generateChart(cfg)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate chart: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(fmt.Sprintf("✅ Generated chart at: %s\n", outputDir))); err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

func generateChart(cfg Config) (string, error) {
	timestamp := time.Now().Unix()
	outputDir := fmt.Sprintf("output/%s-%d", cfg.UmbrellaChartName, timestamp)

	// Copy umbrella chart
	if err := copyChartTemplate("./templates/umbrella", filepath.Join(outputDir, cfg.UmbrellaChartName), map[string]string{
		"umbrella-chart": cfg.UmbrellaChartName,
	}); err != nil {
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
			return "", fmt.Errorf("failed to copy subchart %s: %w", sub.Name, err)
		}

		if err := pruneTemplates(subPath, sub.Workload); err != nil {
			return "", fmt.Errorf("failed to prune templates for %s: %w", sub.Name, err)
		}
	}

	return outputDir, nil
}

func copyChartTemplate(src, dst string, replacements map[string]string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(src, path)
		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		replaced := string(content)
		for old, new := range replacements {
			replaced = strings.ReplaceAll(replaced, old, new)
		}

		return os.WriteFile(targetPath, []byte(replaced), 0644)
	})
}

func pruneTemplates(chartPath, workload string) error {
	templatesPath := filepath.Join(chartPath, "templates")
	workloadFiles := map[string]bool{
		"deployment.yaml":  workload != "deployment",
		"statefulset.yaml": workload != "statefulset",
		"daemonset.yaml":   workload != "daemonset",
	}

	for filename, shouldDelete := range workloadFiles {
		if shouldDelete {
			toRemove := filepath.Join(templatesPath, filename)
			if err := os.Remove(toRemove); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("error removing %s: %w", filename, err)
			}
		}
	}

	return nil
}
