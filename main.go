package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/playwright-community/playwright-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/yaml.v2"
)

// TestStep represents a single step in the test
type TestStep struct {
	Name   string            `yaml:"name"`
	Action map[string]string `yaml:",inline"`
}

var (
	testSuccess = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "playwright_test_success",
			Help: "Indicates if the Playwright test succeeded (1) or failed (0).",
		},
		[]string{"test_name"},
	)
)

func init() {
	// Register Prometheus metrics
	prometheus.MustRegister(testSuccess)
}

func main() {
	// Read the test configuration file (mounted as ConfigMap)
	configFile := "/app/config/tests.yaml"
	configData, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	// Parse the config data into TestSteps
	var testSteps []TestStep
	err = yaml.Unmarshal(configData, &testSteps)
	if err != nil {
		log.Fatalf("Failed to parse config data: %v", err)
	}

	// Initialize Playwright
	pw, err := playwright.Run()
	if err != nil {
		log.Fatalf("Failed to start Playwright: %v", err)
	}
	defer pw.Stop()

	// Create a new browser context
	browser, err := pw.Chromium.Launch()
	if err != nil {
		log.Fatalf("Failed to launch Chromium: %v", err)
	}
	defer browser.Close()

	context, err := browser.NewContext()
	if err != nil {
		log.Fatalf("Failed to create browser context: %v", err)
	}

	page, err := context.NewPage()
	if err != nil {
		log.Fatalf("Failed to create page: %v", err)
	}

	// Execute the test steps and record metrics
	success := 1.0
	for _, step := range testSteps {
		switch {
		case step.Action["navigate"] != "":
			_, err = page.Goto(step.Action["navigate"])
		case step.Action["input"] != "":
			selector := step.Action["selector"]
			text := resolveEnv(step.Action["text"])
			locator := page.Locator(selector)
			err = locator.Fill(text)
		case step.Action["click"] != "":
			locator := page.Locator(step.Action["click"])
			err = locator.Click()
		}

		if err != nil {
			log.Printf("Failed to execute step '%s': %v", step.Name, err)
			success = 0.0
			break
		}
	}

	// Record the test result in Prometheus
	testSuccess.WithLabelValues("linkedin-login").Set(success)
	fmt.Println("Test execution completed")

	// Expose metrics on /metrics endpoint
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func resolveEnv(value string) string {
	if strings.HasPrefix(value, "env://") {
		return os.Getenv(strings.TrimPrefix(value, "env://"))
	}
	return value
}
