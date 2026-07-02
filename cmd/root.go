package cmd

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/kriipke/chartpress/internal/engine"
)

var (
	configPath   string
	templatesDir string
)

// cliConfig is the chartpress.yaml shape: an engine.Spec with an optional
// rules block (omitted rules fall back to engine.DefaultRules).
type cliConfig struct {
	UmbrellaChartName string            `yaml:"umbrellaChartName"`
	Description       string            `yaml:"description"`
	Subcharts         []engine.Subchart `yaml:"subcharts"`
	Dependencies      []string          `yaml:"dependencies"`
	Rules             *engine.Rules     `yaml:"rules"`
}

func loadSpec(path string) (engine.Spec, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return engine.Spec{}, fmt.Errorf("failed to read config file: %w", err)
	}
	var cfg cliConfig
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return engine.Spec{}, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	rules := engine.DefaultRules()
	if cfg.Rules != nil {
		rules = *cfg.Rules
	}
	return engine.Normalize(engine.Spec{
		UmbrellaChartName: cfg.UmbrellaChartName,
		Description:       cfg.Description,
		Subcharts:         cfg.Subcharts,
		Dependencies:      cfg.Dependencies,
		Rules:             rules,
	}), nil
}

var createCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Generates a new umbrella chart and attaches subcharts",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if configPath == "" {
			configPath = "./chartpress.yaml"
		}
		spec, err := loadSpec(configPath)
		if err != nil {
			return err
		}
		// A name argument overrides the config's umbrellaChartName.
		if len(args) == 1 {
			spec = engine.Normalize(engine.Spec{
				UmbrellaChartName: args[0],
				Description:       spec.Description,
				Subcharts:         spec.Subcharts,
				Dependencies:      spec.Dependencies,
				Rules:             spec.Rules,
			})
		}
		return runCreate(spec)
	},
}

// runCreate generates the chart through the same engine the server and
// operator use, then expands packaged subcharts into editable directories.
func runCreate(spec engine.Spec) error {
	outputRoot := fmt.Sprintf("output/%s-%d", spec.UmbrellaChartName, time.Now().Unix())
	chartDir, err := engine.GenerateChart(spec, templatesDir, outputRoot)
	if err != nil {
		return err
	}
	if err := engine.ExpandSubcharts(chartDir); err != nil {
		return err
	}
	for _, w := range engine.Warnings(spec) {
		fmt.Printf("⚠️  %s\n", w)
	}
	fmt.Printf("✅ Generated chart at: %s\n", chartDir)
	return nil
}

var rootCmd = &cobra.Command{
	Use:   "chartpress",
	Short: "CLI tool to define an umbrella Helm chart",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.SetFlags(0)
		log.Fatal(err)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "Path to the configuration YAML file (default: ./chartpress.yaml)")

	rootCmd.AddCommand(createCmd)

	createCmd.Flags().StringVar(&templatesDir, "templates", "./templates", "Path to the chart templates directory (containing umbrella/ and subchart/)")
}
