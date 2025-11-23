package scraper_test

import (
	"io"
	"log/slog"
	"regexp"
	"testing"

	"github.com/libops/cap/config"
	"github.com/libops/cap/scraper"
	"github.com/prometheus/prometheus/storage"
)

type MockScraper struct {
	*scraper.Scraper // Embed the real Scraper to satisfy method calls if needed
	MockConfig       config.Config
}

func NewMockScraper(t *testing.T, pattern string) *MockScraper {
	cfg := config.Config{
		ProjectID: "p",
		Location:  "l",
		Cluster:   "c",
	}

	// Compile the regex for the mock config
	cfg.FilterRegex = regexp.MustCompile(pattern)

	// Create a minimal real Scraper instance (must provide a valid io.Writer for klog)
	s, err := scraper.NewScraper(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), io.Discard)
	if err != nil {
		t.Fatalf("Failed to initialize real scraper for mocking: %v", err)
	}

	return &MockScraper{
		Scraper:    s,
		MockConfig: cfg,
	}
}

// TestProcessBody_Filtering verifies that the filtering logic correctly excludes
// metrics based on container name, metric name, and value.
func TestProcessBody_Filtering(t *testing.T) {
	// Sample Prometheus text format output (simulating cAdvisor)
	sampleBody := `
# HELP container_cpu_usage_seconds_total Cumulative cpu time consumed in seconds.
# TYPE container_cpu_usage_seconds_total counter
container_cpu_usage_seconds_total{id="/",name="cap",namespace="default"} 100.0 1678886400000
container_cpu_usage_seconds_total{id="/kubepods/burstable/pod1/c1",name="my-app",namespace="default"} 5.0 1678886400000
container_cpu_usage_seconds_total{id="/kubepods/burstable/pod1/c2",name="libops-cache",namespace="default"} 1.0 1678886400000
container_cpu_usage_seconds_total{id="/kubepods/burstable/pod1/c3",name="other-app",namespace="default"} 0.0 1678886400000
# HELP container_tasks_state The state of the container's tasks.
# TYPE container_tasks_state gauge
container_tasks_state{state="running",name="my-app"} 1.0 1678886400000
`
	// The regex pattern for this test is set to match all strings (.*)
	mock := NewMockScraper(t, ".*")

	// Overwrite the embedded scraper's config with the mock config for this test
	mock.Cfg = mock.MockConfig

	// Call the new public method
	batch, _, err := mock.ProcessBody([]byte(sampleBody))

	if err != nil {
		t.Fatalf("ProcessBody failed: %v", err)
	}

	// Expected results:
	// 1. "cap" container (name="cap") -> EXCLUDED
	// 2. "my-app" container (name="my-app", metric="container_cpu_usage_seconds_total", val=5.0) -> INCLUDED
	// 3. "libops-cache" container (name="libops-cache") -> EXCLUDED (HasPrefix)
	// 4. "other-app" container (name="other-app", val=0.0) -> EXCLUDED (val <= 0.0)
	// 5. "my-app" tasks_state (metric="container_tasks_state") -> EXCLUDED (Metric name check)

	if len(batch) != 1 {
		t.Fatalf("Expected 1 metric sample after filtering, got %d", len(batch))
	}

	// Helper to find the original labels from the Ref
	// Call the new public method
	labelsByRef := mock.GetLabelsByRef(storage.SeriesRef(batch[0].Ref))
	if labelsByRef.Get("name") != "my-app" {
		t.Errorf("Included metric has wrong container name. Expected 'my-app', got '%s'", labelsByRef.Get("name"))
	}
	if batch[0].V != 5.0 {
		t.Errorf("Included metric has wrong value. Expected 5.0, got %f", batch[0].V)
	}
}

// TestProcessBody_MetricParsing verifies that basic parsing of a single metric works.
func TestProcessBody_MetricParsing(t *testing.T) {
	sampleBody := `
# HELP container_memory_working_set_bytes Current working set of the container.
# TYPE container_memory_working_set_bytes gauge
container_memory_working_set_bytes{id="/",name="test-mem",namespace="test-ns"} 1000000.0
`
	mock := NewMockScraper(t, ".*")
	mock.Cfg = mock.MockConfig

	// Call the new public method
	batch, metadata, err := mock.ProcessBody([]byte(sampleBody))

	if err != nil {
		t.Fatalf("ProcessBody failed: %v", err)
	}

	if len(batch) != 1 {
		t.Fatalf("Expected 1 metric sample, got %d", len(batch))
	}

	meta, ok := metadata["container_memory_working_set_bytes"]
	if !ok {
		t.Fatal("Metadata not found for expected metric")
	}
	if meta.Type != "gauge" {
		t.Errorf("Expected metric type 'gauge', got %s", meta.Type)
	}

	if batch[0].V != 1000000.0 {
		t.Errorf("Expected value 1000000.0, got %f", batch[0].V)
	}

	labelsByRef := mock.GetLabelsByRef(storage.SeriesRef(batch[0].Ref))
	if labelsByRef.Get("name") != "test-mem" {
		t.Errorf("Expected label 'name'='test-mem', got '%s'", labelsByRef.Get("name"))
	}
}
