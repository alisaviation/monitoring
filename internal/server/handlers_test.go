package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/alisaviation/monitoring/internal/models"
	"github.com/alisaviation/monitoring/internal/server"
	"github.com/alisaviation/monitoring/internal/storage"
)

// ExampleServer_UpdateMetrics demonstrates how to update a single metric via JSON request.
// It shows the basic usage of the UpdateMetrics endpoint with a gauge metric.
func ExampleServer_UpdateMetrics() {
	// Create test server with in-memory storage
	srv := server.NewServer(storage.NewMemStorage(""), nil)

	// Prepare gauge metric update
	metric := models.Metric{
		ID:    "test_gauge",
		MType: models.Gauge,
		Value: ptrFloat64(123.45),
	}

	// Marshal metric to JSON and create HTTP request
	body, _ := json.Marshal(metric)
	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	w := httptest.NewRecorder()
	srv.UpdateMetrics(w, req)

	// Print response status code
	fmt.Println("Status:", w.Code)

	// Output:
	// Status: 200
}

// ExampleServer_GetValue demonstrates how to retrieve a metric value via JSON request.
// It shows querying an existing gauge metric from storage.
func ExampleServer_GetValue() {
	// Initialize storage with test data
	memStorage := storage.NewMemStorage("")
	memStorage.SetGauge(context.Background(), "existing_gauge", 56.78)

	srv := server.NewServer(memStorage, nil)

	// Prepare metric query
	metric := models.Metric{
		ID:    "existing_gauge",
		MType: models.Gauge,
	}

	// Marshal request and set headers
	body, _ := json.Marshal(metric)
	req := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	w := httptest.NewRecorder()
	srv.GetValue(w, req)

	// Print response details
	fmt.Println("Status:", w.Code)
	fmt.Println("Body:", w.Body.String())

	// Output:
	// Status: 200
	// Body: {"id":"existing_gauge","type":"gauge","value":56.78}
}

// ExampleServer_UpdateBatchMetrics demonstrates batch updating of multiple metrics.
// It shows how to send both gauge and counter metrics in a single request.
func ExampleServer_UpdateBatchMetrics() {
	srv := server.NewServer(storage.NewMemStorage(""), nil)

	// Prepare batch of metrics to update
	metrics := []models.Metric{
		{
			ID:    "batch_gauge",
			MType: models.Gauge,
			Value: ptrFloat64(99.99),
		},
		{
			ID:    "batch_counter",
			MType: models.Counter,
			Delta: ptrInt64(10),
		},
	}

	// Create JSON request with metrics batch
	body, _ := json.Marshal(metrics)
	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Execute batch update
	w := httptest.NewRecorder()
	srv.UpdateBatchMetrics(w, req)

	// Print response status
	fmt.Println("Status:", w.Code)

	// Output:
	// Status: 200
}

// ExampleServer_GetMetricsList demonstrates retrieving the HTML metrics list page.
// It shows how to get all stored metrics in HTML format.
func ExampleServer_GetMetricsList() {
	// Initialize storage with test data
	memStorage := storage.NewMemStorage("")
	memStorage.SetGauge(context.Background(), "list_gauge", 12.34)
	memStorage.AddCounter(context.Background(), "list_counter", 56)

	srv := server.NewServer(memStorage, nil)

	// Create request for metrics list
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	// Execute request
	w := httptest.NewRecorder()
	srv.GetMetricsList(w, req)

	// Print response status
	fmt.Println("Status:", w.Code)

	// Output:
	// Status: 200
}

// ExampleServer_PingHandler demonstrates the database ping endpoint.
// Note: This example requires a real database connection to work properly.
func ExampleServer_PingHandler() {
	// Needs the real DB
	// Output: Database ping handler
}

// ptrFloat64 is a helper function that returns a pointer to a float64 value.
// Used in tests to create pointer values for metric fields.
func ptrFloat64(f float64) *float64 { return &f }

// ptrInt64 is a helper function that returns a pointer to an int64 value.
// Used in tests to create pointer values for metric fields.
func ptrInt64(i int64) *int64 { return &i }
