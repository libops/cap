# cAdvisor Plus (cap)

[ops-agent](https://docs.cloud.google.com/stackdriver/docs/solutions/agents/ops-agent) metrics for Google Compute Engine instances running Google Container Optimized OS.

This service scrapes cadvisor every 30s. filters out metrics, and sends the unfiltered metrics to Google Cloud Monitoring

## Attribution

Code essentially forked from https://github.com/GoogleCloudPlatform/prometheus-engine/blob/7ca3c5c4e2a43cad2e79edbf724bc58cf064e423/pkg/export/gcm/promtest/local_export.go
