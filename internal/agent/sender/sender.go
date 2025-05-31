package sender

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"

	"github.com/alisaviation/monitoring/internal/logger"
	"github.com/alisaviation/monitoring/internal/models"
)

type Sender struct {
	serverAddress string
	client        *resty.Client
}

func NewSender(serverAddress string) *Sender {
	client := resty.New()
	client.SetHeader("Accept-Encoding", "gzip")
	return &Sender{
		serverAddress: serverAddress,
		client:        client,
	}
}

func (s *Sender) compressData(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(data); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *Sender) SendMetrics(metrics map[string]*models.Metric) {
	var metricsData models.Metric
	for name, metric := range metrics {

		metricsData.ID = name
		metricsData.MType = metric.MType

		if metric.MType == models.Gauge {
			metricsData.Value = metric.Value
		}
		if metric.MType == models.Counter {
			metricsData.Delta = metric.Delta
		}

		jsonData, err := json.Marshal(metricsData)
		if err != nil {
			logger.Log.Error("Error marshaling JSON", zap.Error(err))
			continue
		}
		s.sender(jsonData)
	}
}

func (s *Sender) GetMetric(metric *models.Metric) (*models.Metric, error) {
	jsonData, err := json.Marshal(metric)
	if err != nil {
		logger.Log.Error("Error marshaling JSON", zap.Error(err))
		return nil, err
	}

	compressedData, err := s.compressData(jsonData)
	if err != nil {
		logger.Log.Error("Error compressing data", zap.Error(err))
		return nil, err
	}

	resp, err := s.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		SetBody(compressedData).
		Post("http://" + s.serverAddress + "/value/")
	if err != nil {
		logger.Log.Error("Error sending request", zap.Error(err))
		return nil, err
	}

	if resp != nil {
		defer resp.RawResponse.Body.Close()
	}

	if resp.StatusCode() != http.StatusOK {
		logger.Log.Error("Error response from server", zap.String("status", resp.Status()))
	}

	var receivedMetric models.Metric
	if err := json.Unmarshal(resp.Body(), &receivedMetric); err != nil {
		logger.Log.Error("Error unmarshaling JSON response", zap.Error(err))
		return nil, err
	}

	return &receivedMetric, nil
}

func (s *Sender) SendMetricsBatch(metrics map[string]*models.Metric) {
	if len(metrics) == 0 {
		logger.Log.Error("Error, the batch is empty")
		return
	}

	var metricsList []models.Metric
	for name, metric := range metrics {
		batchMetrics := models.Metric{
			ID:    name,
			MType: metric.MType,
		}
		if metric.MType == models.Gauge {
			batchMetrics.Value = metric.Value
		}
		if metric.MType == models.Counter {
			batchMetrics.Delta = metric.Delta
		}
		metricsList = append(metricsList, batchMetrics)
	}

	jsonData, err := json.Marshal(metricsList)
	if err != nil {
		logger.Log.Error("Error marshaling JSON", zap.Error(err))
		return
	}
	s.sender(jsonData)
}
func (s *Sender) sender(jsonData []byte) {
	compressedData, err := s.compressData(jsonData)
	if err != nil {
		logger.Log.Error("Error compressing data", zap.Error(err))
		return
	}

	resp, err := s.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		SetBody(compressedData).
		Post("http://" + s.serverAddress + "/updates/")
	if err != nil {
		logger.Log.Error("Error sending request", zap.Error(err))
		return
	}

	if resp != nil {
		defer resp.RawResponse.Body.Close()
	}

	if resp.StatusCode() != http.StatusOK {
		logger.Log.Error("Error response from server", zap.String("status", resp.Status()))
	}

}
