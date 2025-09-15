package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/mabulgu/pawdy/internal/app"
	"github.com/spf13/cobra"
)

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check health of all services",
	Long: `Check the health status of all Pawdy services including the LLM backend, 
vector database, embedding service, and safety gate. Reports connection status 
and response times.`,
	RunE: runHealth,
}

func init() {
	rootCmd.AddCommand(healthCmd)
}

func runHealth(cmd *cobra.Command, args []string) error {
	// Initialize the application
	pawdy, err := app.New()
	if err != nil {
		return fmt.Errorf("failed to initialize Pawdy: %w", err)
	}
	defer pawdy.Close()

	fmt.Println("🏥 Pawdy Health Check")
	fmt.Println("═══════════════════")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Check all services
	healthStatus, err := pawdy.HealthCheck(ctx)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	overallHealthy := true
	for _, status := range healthStatus {
		icon := "✅"
		if !status.Healthy {
			icon = "❌"
			overallHealthy = false
		}

		fmt.Printf("%s %s", icon, status.Name)
		
		if status.Latency != "" {
			fmt.Printf(" (%s)", status.Latency)
		}
		
		if status.Message != "" {
			fmt.Printf(" - %s", status.Message)
		}
		
		fmt.Println()
	}

	fmt.Println()
	
	if overallHealthy {
		fmt.Println("🎉 All services are healthy!")
	} else {
		fmt.Println("⚠️  Some services are experiencing issues")
		return fmt.Errorf("health check failed")
	}

	return nil
}
