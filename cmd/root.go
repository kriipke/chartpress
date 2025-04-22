package cmd

import (
    "fmt"
    "io/ioutil"
    "os"
    "path/filepath"
    "strings"
    "encoding/json"
    "github.com/spf13/cobra"

    "helm.sh/helm/v3/pkg/chart"
    "helm.sh/helm/v3/pkg/chart/loader"
    "helm.sh/helm/v3/pkg/chartutil"
    "gopkg.in/yaml.v2"
)

var configPath string
var umbrellaChartPath string
var subchartChartPath string

func loadUmbrellaChart(path string) (*chart.Chart, error) {
    ch, err := loader.Load(path)
    if err != nil {
        return nil, fmt.Errorf("failed to load chart: %w", err)
    }
    return ch, nil
}

// inspectChart provides a detailed overview of the given Helm chart.
func inspectChart(ch *chart.Chart) {
    fmt.Println("=== Chart Metadata ===")
    if ch.Metadata != nil {
        metadataBytes, err := json.MarshalIndent(ch.Metadata, "", "  ")
        if err != nil {
            fmt.Printf("Error marshaling metadata: %v\n", err)
        } else {
            fmt.Println(string(metadataBytes))
        }
    } else {
        fmt.Println("No metadata available.")
    }

    fmt.Println("\n=== Templates ===")
    if len(ch.Templates) > 0 {
        for _, tmpl := range ch.Templates {
            fmt.Printf("- %s (%d bytes)\n", tmpl.Name, len(tmpl.Data))
        }
    } else {
        fmt.Println("No templates found.")
    }

    fmt.Println("\n=== Values ===")
    if len(ch.Values) > 0 {
        valuesBytes, err := json.MarshalIndent(ch.Values, "", "  ")
        if err != nil {
            fmt.Printf("Error marshaling values: %v\n", err)
        } else {
            fmt.Println(string(valuesBytes))
        }
    } else {
        fmt.Println("No values defined.")
    }

    fmt.Println("\n=== Files ===")
    if len(ch.Files) > 0 {
        for _, file := range ch.Files {
            fmt.Printf("- %s (%d bytes)\n", file.Name, len(file.Data))
        }
    } else {
        fmt.Println("No additional files found.")
    }

    fmt.Println("\n=== Dependencies ===")
    if len(ch.Dependencies()) > 0 {
        for _, dep := range ch.Dependencies() {
            fmt.Printf("- %s (version: %s)\n", dep.Metadata.Name, dep.Metadata.Version)
        }
    } else {
        fmt.Println("No dependencies found.")
    }

    fmt.Println("\n=== CRDs ===")
    crds := ch.CRDObjects()
    if len(crds) > 0 {
        for _, crd := range crds {
            fmt.Printf("- %s (%d bytes)\n", crd.Name, len(crd.File.Data))
        }
    } else {
        fmt.Println("No CRDs found.")
    }

    fmt.Println("\n=== Schema ===")
    if len(ch.Schema) > 0 {
        fmt.Printf("%s\n", string(ch.Schema))
    } else {
        fmt.Println("No schema defined.")
    }
}

func listSubcharts(ch *chart.Chart) {
    fmt.Println("Subcharts:")
    for _, subchart := range ch.Dependencies() {
        fmt.Printf("- %s (version: %s)\n", subchart.Metadata.Name, subchart.Metadata.Version)
    }
}

func loadConfig(path string) (*Config, error) {
    data, err := ioutil.ReadFile(path)
    if err != nil {
        return nil, err
    }
    var cfg Config
    if err := yaml.Unmarshal(data, &cfg); err != nil {
        return nil, err
    }
    return &cfg, nil
}

type Config struct {
    UmbrellaChartName string    `yaml:"umbrellaChartName"`
    Subcharts         []Subchart `yaml:"subcharts"`
}

type Subchart struct {
    Name     string `yaml:"name"`
    Workload string `yaml:"workload"`
}


var rootCmd = &cobra.Command{
    Use:   "umbrella-cli",
    Short: "CLI tool to define an umbrella Helm chart",
    RunE: func(cmd *cobra.Command, args []string) error {
        cfg, err := loadConfig(configPath)
        if err != nil {
            return fmt.Errorf("failed to load config: %w", err)
        }

        // Create subcharts based on the configuration
        var subcharts []*chart.Chart
        for _, sc := range cfg.Subcharts {
            subchart := &chart.Chart{
                Metadata: &chart.Metadata{
                    Name:        sc.Name,
                    Version:     "0.1.0",
                    Description: fmt.Sprintf("%s service", sc.Name),
                    APIVersion:  "v2",
                },
                Values: map[string]interface{}{
                    "workload": sc.Workload,
                },
            }
            subcharts = append(subcharts, subchart)
        }

        // Create the umbrella chart
        umbrellaChart := &chart.Chart{
            Metadata: &chart.Metadata{
                Name:        cfg.UmbrellaChartName,
                Version:     "1.0.0",
                Description: "An umbrella chart aggregating multiple subcharts",
                APIVersion:  "v2",
            },
            Values: map[string]interface{}{
                "global": map[string]interface{}{
                    "imagePullPolicy": "IfNotPresent",
                    "replicas":        3,
                },
            },
        }

        // Set dependencies using the provided method
        umbrellaChart.SetDependencies(subcharts...)

        // For demonstration, print the chart's metadata
        fmt.Printf("Umbrella Chart: %s\n", umbrellaChart.Metadata.Name)
        fmt.Printf("Version: %s\n", umbrellaChart.Metadata.Version)
        fmt.Printf("Description: %s\n", umbrellaChart.Metadata.Description)

        return nil
    },
}

func init() {
    rootCmd.Flags().StringVarP(&configPath, "config", "c", "config.yaml", "Path to the configuration YAML file")
    rootCmd.Flags().StringVarP(&umbrellaChartPath, "umbrella-base", "u", "templates/umbrella", "Path to the Umbrella base chart")
    rootCmd.Flags().StringVarP(&subchartChartPath, "subchart-base", "s", "templates/subchart", "Path to the Subchart base chart")
    inspect(umbrellaChartPath)
    inspect(subchartChartPath)
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}


func inspect(path string) {
    chart, err := loadUmbrellaChart(path)
    if err != nil {
        fmt.Println("Error:", err)
        return
    }

    inspectChart(chart)
    listSubcharts(chart)
    test()
}

// cloneAndRenameChart loads a base Helm chart, renames it, updates all references,
// and writes the modified chart to a new directory.
func cloneAndRenameChart(baseChartPath, newChartName, outputDir string) error {
    // Load the base chart
    baseChart, err := loader.Load(baseChartPath)
    if err != nil {
        return fmt.Errorf("failed to load base chart: %w", err)
    }

    // Store the original chart name
    originalName := baseChart.Metadata.Name

    // Update the chart's metadata
    baseChart.Metadata.Name = newChartName

    // Update references in templates
    for _, tmpl := range baseChart.Templates {
        tmpl.Data = []byte(strings.ReplaceAll(string(tmpl.Data), originalName, newChartName))
    }

    // Update references in values.yaml
    if baseChart.Values != nil {
        valuesYAML, err := yaml.Marshal(baseChart.Values)
        if err != nil {
            return fmt.Errorf("failed to marshal values.yaml: %w", err)
        }
        updatedValues := strings.ReplaceAll(string(valuesYAML), originalName, newChartName)
        var newValues map[string]interface{}
        if err := yaml.Unmarshal([]byte(updatedValues), &newValues); err != nil {
            return fmt.Errorf("failed to unmarshal updated values.yaml: %w", err)
        }
        baseChart.Values = newValues
    }

    // Update references in additional files
    for _, file := range baseChart.Files {
        file.Data = []byte(strings.ReplaceAll(string(file.Data), originalName, newChartName))
    }

    // Define the output path
    newChartPath := filepath.Join(outputDir, newChartName)

    // Ensure the output directory exists
    if err := os.MkdirAll(newChartPath, 0755); err != nil {
        return fmt.Errorf("failed to create output directory: %w", err)
    }

    // Save the modified chart
    if err := chartutil.SaveDir(baseChart, newChartPath); err != nil {
        return fmt.Errorf("failed to save modified chart: %w", err)
    }

    fmt.Printf("Modified chart saved to: %s\n", newChartPath)
    return nil
}
func test() {
    baseChartPath := "templates/umbrella"
    newChartName := "new-umbrella"
    outputDir := "output-new"

    if err := cloneAndRenameChart(baseChartPath, newChartName, outputDir); err != nil {
        fmt.Println("Error:", err)
    }
}
