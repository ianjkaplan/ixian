package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	baseURL   string
	output    string
	headers   []string
	xAPIKey   string
	authToken string
)

var rootCmd = &cobra.Command{
	Use:   "generated",
	Short: "CLI for generated API",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&baseURL, "base-url", "https://api.petstore.example.com/v1", "API base URL")
	rootCmd.PersistentFlags().StringVarP(&output, "output", "o", "json", "Output format (json, raw)")
	rootCmd.PersistentFlags().StringArrayVarP(&headers, "header", "H", nil, "Custom headers in key:value format (repeatable)")
	rootCmd.PersistentFlags().StringVar(&xAPIKey, "x-api-key", "", "API key (sent as X-API-Key header)")
	rootCmd.PersistentFlags().StringVar(&authToken, "auth-token", "", "Bearer authentication token")
}
