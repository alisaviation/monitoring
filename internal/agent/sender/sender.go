package sender

import (
	"fmt"
	"net/http"

	"github.com/go-resty/resty/v2"

	"github.com/alisaviation/monitoring/internal/models"
)

type Sender struct {
	serverAddress string
	client        *resty.Client
}

func NewSender(serverAddress string) *Sender {
	return &Sender{
		serverAddress: serverAddress,
		client:        resty.New(),
	}
}

func (s *Sender) SendMetrics(metrics map[string]*models.Metric) {
	for name, metric := range metrics {
		resp, err := s.client.R().
			SetHeader("Content-Type", "text/plain").
			SetPathParams(map[string]string{
				"type":  string(metric.Type),
				"name":  name,
				"value": fmt.Sprintf("%v", metric.Value),
			}).
			Post("http://" + s.serverAddress + "/update/{type}/{name}/{value}")
		if err != nil {
			fmt.Println("Error sending request:", err)
			continue
		}

		if resp != nil {
			defer resp.RawResponse.Body.Close()
			if err := resp.RawResponse.Body.Close(); err != nil {
				fmt.Println("Error closing response body:", err)
			}
		}

		if resp.StatusCode() != http.StatusOK {
			fmt.Println("Error response from server:", resp.Status())
		}
	}
}
