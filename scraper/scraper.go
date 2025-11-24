package scraper

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/prometheus-engine/pkg/export"
	klog "github.com/go-kit/log"
	capConfig "github.com/libops/cap/config" // Adjust import path
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/textparse"
	"github.com/prometheus/prometheus/model/timestamp"
	"github.com/prometheus/prometheus/storage"
	"github.com/prometheus/prometheus/tsdb/chunks"
	"github.com/prometheus/prometheus/tsdb/record"
)

type Scraper struct {
	Cfg         capConfig.Config
	exporter    *export.Exporter
	client      *http.Client
	cAdvisorURL string
	logger      *slog.Logger
	labelsByRef map[storage.SeriesRef]labels.Labels
}

func NewTestScraper(cfg capConfig.Config, logger *slog.Logger) *Scraper {
	return &Scraper{
		Cfg:         cfg,
		logger:      logger,
		labelsByRef: make(map[storage.SeriesRef]labels.Labels),
	}
}

func NewScraper(cfg capConfig.Config, logger *slog.Logger, w io.Writer) (*Scraper, error) {
	e, err := export.New(klog.NewJSONLogger(w), prometheus.NewRegistry(), export.ExporterOpts{
		UserAgentEnv:     "libops-cap",
		Endpoint:         "monitoring.googleapis.com:443",
		Compression:      "none",
		MetricTypePrefix: export.MetricTypePrefix,

		Cluster:   cfg.Cluster,
		Location:  cfg.Location,
		ProjectID: cfg.ProjectID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Prometheus exporter: %w", err)
	}

	s := &Scraper{
		Cfg:         cfg,
		exporter:    e,
		client:      &http.Client{Timeout: 10 * time.Second},
		cAdvisorURL: fmt.Sprintf("http://%s/metrics", cfg.CADVISORHost),
		logger:      logger,
		labelsByRef: make(map[storage.SeriesRef]labels.Labels),
	}

	if err := s.exporter.ApplyConfig(&config.DefaultConfig); err != nil {
		return nil, fmt.Errorf("failed to apply config to exporter: %w", err)
	}
	s.exporter.SetLabelsByIDFunc(s.GetLabelsByRef)

	return s, nil
}

func (s *Scraper) GetLabelsByRef(ref storage.SeriesRef) labels.Labels {
	return s.labelsByRef[ref]
}

func (s *Scraper) Run(ctx context.Context) {
	// Start the background exporter process
	go func() {
		if err := s.exporter.Run(ctx); err != nil {
			s.logger.Error("Exporter failed", "err", err)
		}
	}()

	ticker := time.NewTicker(s.Cfg.ScrapeInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Shutting down scraper", "reason", ctx.Err())
			return
		case <-ticker.C:
			if err := s.scrapeAndExport(); err != nil {
				s.logger.Error("Failed scrape and export iteration", "err", err)
			}
		}
	}
}

// scrapeAndExport performs a single fetch, parse, filter, and export cycle.
func (s *Scraper) scrapeAndExport() error {
	resp, err := s.client.Get(s.cAdvisorURL)
	if err != nil {
		return fmt.Errorf("failed to fetch Prometheus metrics from %s: %w", s.cAdvisorURL, err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			s.logger.Error("Failed to close response body", "err", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("prometheus responded with non-OK status: %s", resp.Status)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	batch, metadata, err := s.ProcessBody(bodyBytes)
	if err != nil {
		return fmt.Errorf("failed to process scraped body: %w", err)
	}

	s.exporter.Export(func(metric string) (export.MetricMetadata, bool) {
		m, ok := metadata[metric]
		return m, ok
	}, batch, nil)

	return nil
}

func (s *Scraper) ProcessBody(bodyBytes []byte) ([]record.RefSample, map[string]export.MetricMetadata, error) {
	tp, err := textparse.New(bodyBytes, "text/plain")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize text parser: %w", err)
	}

	var (
		currMeta export.MetricMetadata
		batch    []record.RefSample
		metadata = map[string]export.MetricMetadata{}
	)

	s.labelsByRef = make(map[storage.SeriesRef]labels.Labels)

	for {
		et, err := tp.Next()
		if et == textparse.EntryInvalid || err != nil {
			break // End of input or parse error
		}

		switch et {
		case textparse.EntryType:
			_, currMeta.Type = tp.Type()
			continue
		case textparse.EntryHelp:
			mName, mHelp := tp.Help()
			currMeta.Metric, currMeta.Help = string(mName), string(mHelp)
			continue
		case textparse.EntryUnit, textparse.EntryComment:
			continue
		case textparse.EntryHistogram:
			// Handle as necessary or skip
			s.logger.Warn("Skipping EntryHistogram (not implemented)", "metric", currMeta.Metric)
			continue
		default:
		}

		t := timestamp.FromTime(time.Now())
		_, parsedTimestamp, val := tp.Series()
		if parsedTimestamp != nil {
			t = *parsedTimestamp
		}

		lset := labels.New()
		// Metric name is already stored in currMeta, we only need the labels populated in lset
		_ = tp.Metric(&lset)
		ref := lset.Hash()

		metadata[currMeta.Metric] = currMeta
		s.labelsByRef[storage.SeriesRef(ref)] = lset

		containerName := lset.Get("name")

		isLibopsContainer := strings.HasPrefix(containerName, "libops-")
		isTasksState := currMeta.Metric == "container_tasks_state"
		isPositiveValue := val > 0.0
		isCapContainer := containerName == "cap"
		matchesRegex := s.Cfg.FilterRegex.MatchString(lset.String())

		if !isLibopsContainer && !isTasksState && isPositiveValue && !isCapContainer && matchesRegex {
			batch = append(batch, record.RefSample{
				Ref: chunks.HeadSeriesRef(ref),
				V:   val,
				T:   t,
			})
		}
	}
	return batch, metadata, nil
}
