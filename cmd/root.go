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
	Rules             *cliRules         `yaml:"rules"`
}

// cliRules mirrors engine.Rules with pointers so an omitted field in a partial
// chartpress.yaml keeps its engine.DefaultRules() value instead of being reset
// to the zero value (which would silently flip default-true options like
// linked_templates and the generate_* toggles off). Matches the server's
// pointer-based decode.
type cliRules struct {
	Ingress                     *string `yaml:"ingress"`
	CommonAnnotations           *bool   `yaml:"common_annotations"`
	LinkedTemplates             *bool   `yaml:"linked_templates"`
	ResourceNamesMatchChartName *bool   `yaml:"resource_names_match_chart_name"`
	SharedSecretsConfig         *bool   `yaml:"shared_secrets_config"`
	SharedNewrelicConfig        *bool   `yaml:"shared_newrelic_config"`
	GenerateUmbrellaReadme      *bool   `yaml:"generate_umbrella_readme"`
	GenerateSubchartReadme      *bool   `yaml:"generate_subchart_readme"`
	IncludeDocs                 *bool   `yaml:"include_docs"`
	GenerateHandoff             *bool   `yaml:"generate_handoff"`
}

// mergeRules overlays the config's explicitly-set rule fields onto the locked
// defaults, leaving omitted fields at their default value.
func mergeRules(cr *cliRules) engine.Rules {
	rules := engine.DefaultRules()
	if cr == nil {
		return rules
	}
	if cr.Ingress != nil {
		rules.Ingress = *cr.Ingress
	}
	if cr.CommonAnnotations != nil {
		rules.CommonAnnotations = *cr.CommonAnnotations
	}
	if cr.LinkedTemplates != nil {
		rules.LinkedTemplates = *cr.LinkedTemplates
	}
	if cr.ResourceNamesMatchChartName != nil {
		rules.ResourceNamesMatchChartName = *cr.ResourceNamesMatchChartName
	}
	if cr.SharedSecretsConfig != nil {
		rules.SharedSecretsConfig = *cr.SharedSecretsConfig
	}
	if cr.SharedNewrelicConfig != nil {
		rules.SharedNewrelicConfig = *cr.SharedNewrelicConfig
	}
	if cr.GenerateUmbrellaReadme != nil {
		rules.GenerateUmbrellaReadme = *cr.GenerateUmbrellaReadme
	}
	if cr.GenerateSubchartReadme != nil {
		rules.GenerateSubchartReadme = *cr.GenerateSubchartReadme
	}
	if cr.IncludeDocs != nil {
		rules.IncludeDocs = *cr.IncludeDocs
	}
	rules.GenerateHandoff = cr.GenerateHandoff // pointer: nil = enabled
	return rules
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
	return engine.Normalize(engine.Spec{
		UmbrellaChartName: cfg.UmbrellaChartName,
		Description:       cfg.Description,
		Subcharts:         cfg.Subcharts,
		Dependencies:      cfg.Dependencies,
		Rules:             mergeRules(cfg.Rules),
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
