package sender

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/alisaviation/monitoring/internal/models"
)

type Sender struct {
	serverAddress string
}

func NewSender(serverAddress string) *Sender {
	return &Sender{
		serverAddress: serverAddress,
	}
}

func (s *Sender) SendMetrics(metrics map[string]models.Metric) {
	for name, metric := range metrics {
		url := fmt.Sprintf("%s/update/%s/%s/%v", s.serverAddress, metric.Type, name, metric.Value)
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer([]byte{}))
		if err != nil {
			fmt.Println("Error creating request:", err)
			continue
		}
		req.Header.Set("Content-Type", "text/plain")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("Error sending request:", err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Println("Error response from server for %s: %s\n", url, resp.Status)
		} else {
			fmt.Println("Metrics sent successfully to %s\n", url)
		}
	}
}
