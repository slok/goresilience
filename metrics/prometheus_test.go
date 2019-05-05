package metrics_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/assert"

	"github.com/slok/goresilience/metrics"
)

func TestPrometheus(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name          string
		recordMetrics func(metrics.Recorder)
		expMetrics    []string
	}{
		{
			name: "Recording command metrics should expose the metrics.",
			recordMetrics: func(m metrics.Recorder) {
				m1 := m.WithID("test")
				m2 := m.WithID("test2")
				m1.ObserveCommandExecution(now.Add(-450*time.Millisecond), true)
				m1.ObserveCommandExecution(now.Add(-50*time.Millisecond), false)
				m1.ObserveCommandExecution(now.Add(-2*time.Second), true)
				m2.ObserveCommandExecution(now.Add(-1200*time.Millisecond), false)
			},
			expMetrics: []string{
				`goresilience_command_execution_duration_seconds_bucket{id="test",success="false",le="0.005"} 0`,
				`goresilience_command_execution_duration_seconds_bucket{id="test",success="false",le="0.01"} 0`,
				`goresilience_command_execution_duration_seconds_bucket{id="test",success="false",le="0.025"} 0`,
				`goresilience_command_execution_duration_seconds_bucket{id="test",success="false",le="0.05"} 0`,
				`goresilience_command_execution_duration_seconds_bucket{id="test",success="false",le="0.1"} 1`,
				`goresilience_command_execution_duration_seconds_bucket{id="test",success="false",le="0.25"} 1`,
				`goresilience_command_execution_duration_seconds_bucket{id="test",success="false",le="0.5"} 1`,
				`goresilience_command_execution_duration_seconds_bucket{id="test",success="false",le="1"} 1`,
				`goresilience_command_execution_duration_seconds_bucket{id="test",success="false",le="2.5"} 1`,
				`goresilience_command_execution_duration_seconds_bucket{id="test",success="false",le="5"} 1`,
				`goresilience_command_execution_duration_seconds_bucket{id="test",success="false",le="10"} 1`,
				`goresilience_command_execution_duration_seconds_bucket{id="test",success="false",le="+Inf"} 1`,
				`goresilience_command_execution_duration_seconds_count{id="test",success="false"} 1`,

				`goresilience_command_execution_duration_seconds_bucket{id="test",success="true",le="0.005"} 0`,
				`goresilience_command_execution_duration_seconds_bucket{id="test",success="true",le="0.01"} 0`,
				`goresilience_command_execution_duration_seconds_bucket{id="test",success="true",le="0.025"} 0`,
				`goresilience_command_execution_duration_seconds_bucket{id="test",success="true",le="0.05"} 0`,
				`goresilience_command_execution_duration_seconds_bucket{id="test",success="true",le="0.1"} 0`,
				`goresilience_command_execution_duration_seconds_bucket{id="test",success="true",le="0.25"} 0`,
				`goresilience_command_execution_duration_seconds_bucket{id="test",success="true",le="0.5"} 1`,
				`goresilience_command_execution_duration_seconds_bucket{id="test",success="true",le="1"} 1`,
				`goresilience_command_execution_duration_seconds_bucket{id="test",success="true",le="2.5"} 2`,
				`goresilience_command_execution_duration_seconds_bucket{id="test",success="true",le="5"} 2`,
				`goresilience_command_execution_duration_seconds_bucket{id="test",success="true",le="10"} 2`,
				`goresilience_command_execution_duration_seconds_bucket{id="test",success="true",le="+Inf"} 2`,
				`goresilience_command_execution_duration_seconds_count{id="test",success="true"} 2`,

				`goresilience_command_execution_duration_seconds_bucket{id="test2",success="false",le="0.005"} 0`,
				`goresilience_command_execution_duration_seconds_bucket{id="test2",success="false",le="0.01"} 0`,
				`goresilience_command_execution_duration_seconds_bucket{id="test2",success="false",le="0.025"} 0`,
				`goresilience_command_execution_duration_seconds_bucket{id="test2",success="false",le="0.05"} 0`,
				`goresilience_command_execution_duration_seconds_bucket{id="test2",success="false",le="0.1"} 0`,
				`goresilience_command_execution_duration_seconds_bucket{id="test2",success="false",le="0.25"} 0`,
				`goresilience_command_execution_duration_seconds_bucket{id="test2",success="false",le="0.5"} 0`,
				`goresilience_command_execution_duration_seconds_bucket{id="test2",success="false",le="1"} 0`,
				`goresilience_command_execution_duration_seconds_bucket{id="test2",success="false",le="2.5"} 1`,
				`goresilience_command_execution_duration_seconds_bucket{id="test2",success="false",le="5"} 1`,
				`goresilience_command_execution_duration_seconds_bucket{id="test2",success="false",le="10"} 1`,
				`goresilience_command_execution_duration_seconds_bucket{id="test2",success="false",le="+Inf"} 1`,
				`goresilience_command_execution_duration_seconds_count{id="test2",success="false"} 1`,
			},
		},

		{
			name: "Recording retry metrics should expose the metrics.",
			recordMetrics: func(m metrics.Recorder) {
				m1 := m.WithID("test")
				m2 := m.WithID("test2")
				m1.IncRetry()
				m1.IncRetry()
				m2.IncRetry()
			},
			expMetrics: []string{
				`goresilience_retry_retries_total{id="test"} 2`,
				`goresilience_retry_retries_total{id="test2"} 1`,
			},
		},
		{
			name: "Recording timeout metrics should expose the metrics.",
			recordMetrics: func(m metrics.Recorder) {
				m1 := m.WithID("test")
				m2 := m.WithID("test2")
				m1.IncTimeout()
				m1.IncTimeout()
				m2.IncTimeout()
			},
			expMetrics: []string{
				`goresilience_timeout_timeouts_total{id="test"} 2`,
				`goresilience_timeout_timeouts_total{id="test2"} 1`,
			},
		},
		{
			name: "Recording bulkhead metrics should expose the metrics.",
			recordMetrics: func(m metrics.Recorder) {
				m1 := m.WithID("test")
				m2 := m.WithID("test2")
				m1.IncBulkheadQueued()
				m1.IncBulkheadQueued()
				m1.IncBulkheadProcessed()
				m2.IncBulkheadTimeout()
			},
			expMetrics: []string{
				`goresilience_bulkhead_processed_total{id="test"} 1`,
				`goresilience_bulkhead_queued_total{id="test"} 2`,
				`goresilience_bulkhead_timeouts_total{id="test2"} 1`,
			},
		},
		{
			name: "Recording circuitbreaker metrics should expose the metrics.",
			recordMetrics: func(m metrics.Recorder) {
				m1 := m.WithID("test")
				m2 := m.WithID("test2")
				m1.IncCircuitbreakerState("open")
				m1.IncCircuitbreakerState("close")
				m2.IncCircuitbreakerState("close")
				m1.IncCircuitbreakerState("close")
				m1.IncCircuitbreakerState("half-open")
			},
			expMetrics: []string{
				`goresilience_circuitbreaker_state_changes_total{id="test",state="half-open"} 1`,
				`goresilience_circuitbreaker_state_changes_total{id="test",state="open"} 1`,
				`goresilience_circuitbreaker_state_changes_total{id="test",state="close"} 2`,
				`goresilience_circuitbreaker_state_changes_total{id="test2",state="close"} 1`,
			},
		},
		{
			name: "Recording circuitbreaker circuit breaker condition should expose the condition.",
			recordMetrics: func(m metrics.Recorder) {
				m1 := m.WithID("test")
				m2 := m.WithID("test2")
				m1.SetCircuitbreakerCurrentCondition(0) // new
				m1.SetCircuitbreakerCurrentCondition(3) // open
				m1.SetCircuitbreakerCurrentCondition(1) // close
				m2.SetCircuitbreakerCurrentCondition(2) // half-open
			},
			expMetrics: []string{
				`goresilience_circuitbreaker_current_condition{id="test"} 1`,
				`goresilience_circuitbreaker_current_condition{id="test2"} 2`,
			},
		},
		{
			name: "Recording chaos metrics should expose the metrics.",
			recordMetrics: func(m metrics.Recorder) {
				m1 := m.WithID("test")
				m2 := m.WithID("test2")
				m1.IncChaosInjectedFailure("latency")
				m1.IncChaosInjectedFailure("error")
				m2.IncChaosInjectedFailure("error")
			},
			expMetrics: []string{
				`goresilience_chaos_failure_injections_total{id="test",kind="error"} 1`,
				`goresilience_chaos_failure_injections_total{id="test",kind="latency"} 1`,
				`goresilience_chaos_failure_injections_total{id="test2",kind="error"} 1`,
			},
		},
		{
			name: "Recording concurrency limit metrics should expose the metrics.",
			recordMetrics: func(m metrics.Recorder) {
				m1 := m.WithID("test")
				m2 := m.WithID("test2")
				m1.SetConcurrencyLimitInflightExecutions(4)
				m2.SetConcurrencyLimitInflightExecutions(6)
				m1.SetConcurrencyLimitLimiterLimit(1987)
				m2.SetConcurrencyLimitLimiterLimit(16)
				m1.IncConcurrencyLimitResult("success")
				m1.IncConcurrencyLimitResult("success")
				m2.IncConcurrencyLimitResult("ignore")
			},
			expMetrics: []string{
				`goresilience_concurrencylimit_inflight_executions{id="test"} 4`,
				`goresilience_concurrencylimit_inflight_executions{id="test2"} 6`,
				`goresilience_concurrencylimit_limiter_limit{id="test"} 1987`,
				`goresilience_concurrencylimit_limiter_limit{id="test2"} 16`,
				`goresilience_concurrencylimit_result_total{id="test",result="success"} 2`,
				`goresilience_concurrencylimit_result_total{id="test2",result="ignore"} 1`,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			reg := prometheus.NewRegistry()
			p := metrics.NewPrometheusRecorder(reg)

			test.recordMetrics(p)

			// Get the metrics handler and serve.
			h := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/metrics", nil)
			h.ServeHTTP(rec, req)

			resp := rec.Result()

			// Check all metrics are present.
			if assert.Equal(http.StatusOK, resp.StatusCode) {
				body, _ := ioutil.ReadAll(resp.Body)
				for _, expMetric := range test.expMetrics {
					assert.Contains(string(body), expMetric, "metric not present on the result of metrics service")
				}
			}
		})
	}
}
