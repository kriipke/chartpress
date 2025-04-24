package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

type Config struct {
	UmbrellaChartName string     `json:"umbrellaChartName" yaml:"umbrellaChartName"`
	Subcharts         []Subchart `json:"subcharts" yaml:"subcharts"`
}

type Subchart struct {
	Name     string `json:"name" yaml:"name"`
	Workload string `json:"workload" yaml:"workload"` // deployment, statefulset, daemonset
}

func SaveAndGenerateChart(cfg Config) (string, error) {
	yamlBytes, err := yaml.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal config to YAML: %w", err)
	}

	if err := os.WriteFile("chartpress.yaml", yamlBytes, 0644); err != nil {
		return "", fmt.Errorf("failed to write config file: %w", err)
	}

	return generateChart(cfg)
}

func generateChart(cfg Config) (string, error) {
	timestamp := time.Now().Unix()
	outputDir := fmt.Sprintf("output/%s-%d", cfg.UmbrellaChartName, timestamp)

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
