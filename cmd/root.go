package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/chart"
)

var rootCmd = &cobra.Command{
	Use:   "umbrella-cli",
	Short: "CLI tool to define an umbrella Helm chart",
	Run: func(cmd *cobra.Command, args []string) {
		// Define the umbrella chart
		umbrellaChart := &chart.Chart{
			Metadata: &chart.Metadata{
				Name:        "umbrella-chart",
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
			Dependencies: []*chart.Chart{
				{
					Metadata: &chart.Metadata{
						Name:        "frontend",
						Version:     "1.2.3",
						Description: "Frontend service",
						APIVersion:  "v2",
					},
					Values: map[string]interface{}{
						"image": map[string]interface{}{
							"repository": "my-app/frontend",
							"tag":        "latest",
						},
						"annotations": map[string]interface{}{
							"reloader.stakater.com/auto": "true",
						},
						"labels": map[string]interface{}{
							"tier": "frontend",
						},
					},
				},
				{
					Metadata: &chart.Metadata{
						Name:        "redis",
						Version:     "6.0.0",
						Description: "Redis service",
						APIVersion:  "v2",
					},
					Values: map[string]interface{}{
						"persistence": map[string]interface{}{
							"enabled": true,
							"size":    "1Gi",
						},
					},
				},
			},
		}

		// For demonstration, print the chart's metadata
		fmt.Printf("Umbrella Chart: %s\n", umbrellaChart.Metadata.Name)
		fmt.Printf("Version: %s\n", umbrellaChart.Metadata.Version)
		fmt.Printf("Description: %s\n", umbrellaChart.Metadata.Description)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

