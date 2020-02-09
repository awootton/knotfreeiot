package promutil

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// to access the metrics curl http://localhost:2112/metrics

var promHandler http.Handler

// StartPromServer starts a web server on a port that serves prometheus data.
func StartPromServer() {

	promHandler = promhttp.Handler()
	http.Handle("/metrics", promHandler)
	http.ListenAndServe(":2112", nil)

}

// GetReport snags the stuff without calling the http handler.
func GetReport() string {

	var sb strings.Builder

	metrics, err := prometheus.DefaultGatherer.Gather() // ([]*dto.MetricFamily, error)
	if err == nil {
		for _, met := range metrics {
			//if strings.HasPrefix(*met.Name, "go_") == false && strings.HasPrefix(*met.Name, "http_") == false {
			//	sb.WriteString(fmt.Sprintln(met))
			//}
			if strings.Contains(*met.Name, "atw") || strings.Contains(*met.Name, "knot") {
				sb.WriteString(fmt.Sprintln(met))
			}
		}
	}
	return sb.String()
}
