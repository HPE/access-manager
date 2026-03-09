/*
 * SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
 */

package metrics

import (
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/hpe/access-manager/pkg/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	GrpcRequestsTotal   *prometheus.CounterVec
	GrpcErrors          *prometheus.CounterVec
	SubscriptionRetries *prometheus.CounterVec
	GrpcLatency         *prometheus.HistogramVec

	SubscriberCount *prometheus.GaugeVec

	GlobalPodValue *prometheus.GaugeVec // TODO: need to add this
}

func NewMetrics() *Metrics {
	m := &Metrics{
		GrpcRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "grpc_requests_total",
				Help: "Total number of gRPC requests",
			},
			[]string{"method"},
		),
		GrpcLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "grpc_latency_duration_seconds",
				Help:    "Latency of gRPC requests in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method"},
		),
		GrpcErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "grpc_errors_total",
				Help: "Total number of gRPC errors",
			},
			[]string{"method"},
		),
		SubscriptionRetries: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "subscription_retries_total",
				Help: "Total number of retries for subscription",
			},
			[]string{"method"},
		),
		SubscriberCount: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "subscriber_count_total",
				Help: "Total number of subscriber",
			},
			[]string{"method"},
		),
		GlobalPodValue: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "global_pod_value",
				Help: "Current global pod value.",
			},
			[]string{"method"},
		),
	}
	prometheus.MustRegister(m.GrpcRequestsTotal)
	prometheus.MustRegister(m.GrpcLatency)
	prometheus.MustRegister(m.GrpcErrors)
	prometheus.MustRegister(m.SubscriberCount)
	prometheus.MustRegister(m.GlobalPodValue)
	prometheus.MustRegister(m.SubscriptionRetries)
	return m
}

func (m *Metrics) StartMetricsServer(port string) {
	logger.GetLogger().Info().Msg("Starting metrics server : " + port)
	http.Handle("/metrics", promhttp.Handler())

	if err := http.ListenAndServe(port, nil); err != nil { //nolint:gosec
		logger.GetLogger().Err(err).Msg("failed to serve metrics server")
	}
}

func (m *Metrics) addGrpcRequestCount(methodName string) {
	m.GrpcRequestsTotal.WithLabelValues(methodName).Inc()
}

func (m *Metrics) AddGrpcErrorCount() {
	methodName := m.getCallerFunctionName()
	m.GrpcErrors.WithLabelValues(methodName).Inc()
}

func (m *Metrics) AddGlobalValue(value uint64) {
	methodName := m.getCallerFunctionName()
	m.GlobalPodValue.WithLabelValues(methodName).Set(float64(value))
}

func (m *Metrics) AddSubscription() {
	methodName := m.getCallerFunctionName()
	m.SubscriberCount.WithLabelValues(methodName).Inc()
}

func (m *Metrics) RemoveSubscription() {
	methodName := m.getCallerFunctionName()
	m.SubscriberCount.WithLabelValues(methodName).Dec()
}

func (m *Metrics) AddSubscriptionRetriedCount() {
	methodName := m.getCallerFunctionName()
	m.SubscriptionRetries.WithLabelValues(methodName).Inc()
}

func (m *Metrics) AddGrpcReqLatency() func() {
	startTime := time.Now()
	methodName := m.getCallerFunctionName()
	m.addGrpcRequestCount(methodName)

	return func() {
		duration := time.Since(startTime).Seconds()
		m.GrpcLatency.WithLabelValues(methodName).Observe(duration)
	}
}

func (m *Metrics) getCallerFunctionName() string {
	methodName := "Unknown"
	pc, _, _, _ := runtime.Caller(2) //nolint:dogsled
	callerFunc := runtime.FuncForPC(pc)
	if callerFunc != nil {
		// Retrieve the name of the caller function
		splitName := strings.Split(callerFunc.Name(), ".")
		methodName = splitName[len(splitName)-1]
	}

	return methodName
}
