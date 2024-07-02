/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package monitoring

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// Start time to measure uptime
var startTime = time.Now()

// Metrics is a map to store created metrics.
var (
	namespace = "backup_operator"
	//  ┐─┐┌┐┐┌─┐┬─┐┬─┐┌─┐┬─┐
	//  └─┐ │ │ ││┬┘│─┤│ ┬├─
	//  ──┘ ┘ ┘─┘┘└┘┘ ┘┘─┘┴─┘
	BackupOperatorStorageStatus = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "storage",
		Name:      "status",
		Help:      "BackupStorage execution status.",
	}, []string{"name", "type"})
	// ┐─┐┌─┐┬ ┬┬─┐┬─┐┬ ┐┬  ┬─┐
	// └─┐│  │─┤├─ │ ││ ││  ├─
	// ──┘└─┘┘ ┴┴─┘┘─┘┘─┘┘─┘┴─┘
	BackupOperatorScheduleStatus = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "schedule",
		Name:      "status",
		Help:      "BackupSchedule execution status.",
	}, []string{"namespace", "name", "storage"})
	// ┬─┐┬ ┐┌┐┐
	// │┬┘│ ││││
	// ┘└┘┘─┘┘└┘
	BackupOperatorRunStatus = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "run",
		Name:      "status",
		Help:      "BackupRun execution status, hold time of last status change.",
	}, []string{"namespace", "name", "state", "schedule", "storage", "path"})
	BackupOperatorRunBackupSizeBytes = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "run",
		Name:      "backup_size_bytes",
		Help:      "Size of data stored in the backup storage.",
	}, []string{"namespace", "name"})
	//  ┌─┐┬─┐┬─┐┬─┐┬─┐┌┐┐┌─┐┬─┐
	//  │ ││─┘├─ │┬┘│─┤ │ │ ││┬┘
	//  ┘─┘┘  ┴─┘┘└┘┘ ┘ ┘ ┘─┘┘└┘
	BackupOperatorUptimeSeconds = prometheus.NewCounterFunc(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "operator",
		Name:      "uptime_seconds",
		Help:      "Operator uptime in seconds",
	}, func() float64 { return time.Since(startTime).Seconds() })
)

// RegisterMetrics registers all metrics in the Metrics map with Prometheus's global registry.
func RegisterMetrics() {
	metrics.Registry.MustRegister(BackupOperatorStorageStatus)
	metrics.Registry.MustRegister(BackupOperatorScheduleStatus)
	metrics.Registry.MustRegister(BackupOperatorRunStatus)
	metrics.Registry.MustRegister(BackupOperatorRunBackupSizeBytes)
	metrics.Registry.MustRegister(BackupOperatorUptimeSeconds)
}
