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
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// Start time to measure uptime
var startTime = time.Now()

// Metrics is a map to store created metrics.
var (
	metricsNamespace = "backup_operator"
	//  ┐─┐┌┐┐┌─┐┬─┐┬─┐┌─┐┬─┐
	//  └─┐ │ │ ││┬┘│─┤│ ┬├─
	//  ──┘ ┘ ┘─┘┘└┘┘ ┘┘─┘┴─┘
	BackupOperatorStorageStatusFullName = fmt.Sprintf("%s_%s_%s", metricsNamespace, "storage", "status")
	BackupOperatorStorageStatus         = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricsNamespace,
		Subsystem: "storage",
		Name:      "status",
		Help:      "BackupStorage execution status.",
	}, []string{"name", "type"})
	// ┐─┐┌─┐┬ ┬┬─┐┬─┐┬ ┐┬  ┬─┐
	// └─┐│  │─┤├─ │ ││ ││  ├─
	// ──┘└─┘┘ ┴┴─┘┘─┘┘─┘┘─┘┴─┘
	BackupOperatorScheduleStatusFullName = fmt.Sprintf("%s_%s_%s", metricsNamespace, "schedule", "status")
	BackupOperatorScheduleStatus         = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricsNamespace,
		Subsystem: "schedule",
		Name:      "status",
		Help:      "BackupSchedule execution status.",
	}, []string{"namespace", "name", "storage"})
	// ┬─┐┬ ┐┌┐┐
	// │┬┘│ ││││
	// ┘└┘┘─┘┘└┘
	BackupOperatorRunStatusFullName = fmt.Sprintf("%s_%s_%s", metricsNamespace, "run", "status")
	BackupOperatorRunStatus         = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricsNamespace,
		Subsystem: "run",
		Name:      "status",
		Help:      "BackupRun execution status, hold time of last status change.",
	}, []string{"namespace", "name", "state", "schedule", "storage", "path"})
	BackupOperatorRunBackupSizeBytesFullName = fmt.Sprintf("%s_%s_%s", metricsNamespace, "run", "backup_size_bytes")
	BackupOperatorRunBackupSizeBytes         = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricsNamespace,
		Subsystem: "run",
		Name:      "backup_size_bytes",
		Help:      "Size of data stored in the backup storage.",
	}, []string{"namespace", "name"})
	//  ┌─┐┬─┐┬─┐┬─┐┬─┐┌┐┐┌─┐┬─┐
	//  │ ││─┘├─ │┬┘│─┤ │ │ ││┬┘
	//  ┘─┘┘  ┴─┘┘└┘┘ ┘ ┘ ┘─┘┘└┘
	BackupOperatorUptimeSecondsFullName = fmt.Sprintf("%s_%s", metricsNamespace, "uptime_seconds")
	BackupOperatorUptimeSeconds         = prometheus.NewCounterFunc(prometheus.CounterOpts{
		Namespace: metricsNamespace,
		Name:      "uptime_seconds",
		Help:      "Operator uptime in seconds",
	}, func() float64 { return time.Since(startTime).Seconds() })
)

//+kubebuilder:rbac:groups=monitoring.coreos.com,resources=prometheusrules,verbs=get;list;watch;create;update;patch;delete

// RegisterMetrics registers all metrics in the Metrics map with Prometheus's global registry.
func RegisterMetrics() {
	metrics.Registry.MustRegister(BackupOperatorStorageStatus)
	metrics.Registry.MustRegister(BackupOperatorScheduleStatus)
	metrics.Registry.MustRegister(BackupOperatorRunStatus)
	metrics.Registry.MustRegister(BackupOperatorRunBackupSizeBytes)
	metrics.Registry.MustRegister(BackupOperatorUptimeSeconds)
}
