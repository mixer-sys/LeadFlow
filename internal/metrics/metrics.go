package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	WebhookDeliveriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "leadflow_webhook_deliveries_total",
			Help: "Total webhook delivery attempts by status",
		},
		[]string{"status"},
	)

	WebhookDeliveryRetriesTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "leadflow_webhook_delivery_retries_total", Help: "Total webhook delivery retries",
		},
	)

	WebhookDeadLettersTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "leadflow_webhook_dead_letters_total",
			Help: "Total webhook deliveries moved to dead-letter",
		},
	)

	WebhookCircuitBreakerOpen = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "leadflow_webhook_circuit_breaker_open",
			Help: "1 if webhook circuit breaker is open, otherwise 0",
		},
	)

	WebhookDeliveryDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "leadflow_webhook_delivery_duration_seconds",
			Help:    "Webhook delivery duration",
			Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10},
		},
	)
)

func ObserveDelivery(status string, durationSeconds float64) {
	WebhookDeliveriesTotal.WithLabelValues(status).Inc()
	WebhookDeliveryDuration.Observe(durationSeconds)
}
