package cmd

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"../.."

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

type Config struct {
	UmbrellaChartName string
	Subcharts         []Subchart
}

type Subchart struct {
	Name     string
	Workload string // deployment, statefulset, or daemonset
}

var (
	configPath       string
	umbrellaTemplate string
	subchartTemplate string
)

func loadConfig(path string) (*Config, error) {
	// Read the file using os.ReadFile
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	// Unmarshal YAML content
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

var createCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Generates a new umbrella chart and attaches subcharts",
	Args:  cobra.ExactArgs(1), // Ensure exactly one argument is provided
	Run: func(cmd *cobra.Command, args []string) {
		chartName := args[0] // Get the chart name from the argument
		runCreate(chartName)
	},
}

func runCreate(chartName string) {
	// Use default config path if none is provided
	if configPath == "" {
		configPath = "./chartpress.yaml"
	}

	// Load config
	configData, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(configData, &cfg); err != nil {
		log.Fatalf("Failed to unmarshal config: %v", err)
	}

	timestamp := time.Now().Unix()
	outputDir := fmt.Sprintf("output/%s-%d", cfg.UmbrellaChartName, timestamp)

	// Copy umbrella chart
	if err := copyChartTemplate(umbrellaTemplate, filepath.Join(outputDir, cfg.UmbrellaChartName), map[string]string{
		"umbrella-chart": cfg.UmbrellaChartName,
	}); err != nil {
		log.Fatalf("Failed to copy umbrella chart: %v", err)
	}

	chartsDir := filepath.Join(outputDir, cfg.UmbrellaChartName, "charts")

	for _, sub := range cfg.Subcharts {
		subPath := filepath.Join(chartsDir, sub.Name)

		replacements := map[string]string{
			"component":      sub.Name,
			"umbrella-chart": cfg.UmbrellaChartName,
		}

		if err := copyChartTemplate(subchartTemplate, subPath, replacements); err != nil {
			log.Fatalf("Failed to copy subchart %s: %v", sub.Name, err)
		}

		if err := pruneTemplates(subPath, sub.Workload); err != nil {
			log.Fatalf("Failed to prune templates for %s: %v", sub.Name, err)
		}
	}

	fmt.Printf("✅ Generated chart at: %s\n", outputDir)
}

// Recursively copies chart template with placeholder replacements
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

// Removes unnecessary workload templates
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

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Runs the chartpress server",
	Run: func(cmd *cobra.Command, args []string) {
		runServer()
	},
}

func runServer() {
	http.HandleFunc("/generate", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Handling /generate...")
		w.Header().Set("Access-Control-Allow-Origin", "*") // Allow all origins
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}
		handleGenerate(w, r)
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

var rootCmd = &cobra.Command{
	Use:   "chartpress [name]",
	Short: "CLI tool to define an umbrella Helm chart",
	Args:  cobra.ExactArgs(1), // Ensure exactly one argument is provided
	RunE: func(cmd *cobra.Command, args []string) error {
		chartName := args[0] // Get the chart name from the argument

		// Use default config path if none is provided
		if configPath == "" {
			configPath = "./chartpress.yaml"
		}

		configData, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("failed to read config file: %w", err)
		}

		var cfg Config
		if err := yaml.Unmarshal(configData, &cfg); err != nil {
			return fmt.Errorf("failed to unmarshal config: %w", err)
		}

		// Add logic here if needed
		fmt.Printf("Chart name: %s\n", chartName)
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to the configuration YAML file (default: ./chartpress.yaml)")

	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(createCmd)

	createCmd.Flags().StringVar(&umbrellaTemplate, "umbrella-template", "./templates/umbrella", "Path to umbrella chart template")
	createCmd.Flags().StringVar(&subchartTemplate, "subchart-template", "./templates/subchart", "Path to subchart template")
}
