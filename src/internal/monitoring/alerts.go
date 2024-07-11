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
	"os"

	prometheus "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	client "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ruleName           = "backup-operator-rules"              // Name of the Prometheus rule
	alertRuleGroup     = "backup-operator.rules"              // Name of the alert rule group
	runbookURLBasePath = "https://backup-operator.io/metrics" // Base URL for runbooks
)

var (
	alertsInitialized = false                                 // Flag indicating if alerts are initialized
	alertsEnabled     = os.Getenv("ENABLE_ALERTS") != "false" // Flag indicating if alerts are enabled based on environment variable
	alertsNamespace   = os.Getenv("ALERTS_NAMESPACE")         // Namespace for alerts based on environment variable
)

// RegisterAlerts registers Prometheus alert rules in the cluster
func RegisterAlerts(ctx context.Context, c client.Client) (err error) {
	log := ctrl.Log.WithName("register-alerts") // Logger for register-alerts

	// Check if alerts are disabled or already initialized
	if !alertsEnabled || alertsInitialized {
		log.Info("alert rules creation is disabled")
		return
	}

	// Determine the namespace for alerts
	if alertsNamespace == "" {
		namespaceFile := "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
		var n []byte
		if n, err = os.ReadFile(namespaceFile); err != nil {
			alertsNamespace = string(n)
		} else {
			err = fmt.Errorf("alert rules namespace is not defined and could not be obtained from serviceAccount file: %s", err.Error())
			return
		}
	}

	// Create or update the Prometheus rule
	rule := NewPrometheusRule(alertsNamespace)
	existingRule := &prometheus.PrometheusRule{}
	if err = c.Get(ctx, types.NamespacedName{Name: ruleName, Namespace: alertsNamespace}, existingRule); err == nil {
		rule.SetResourceVersion(existingRule.GetResourceVersion())
		if err = c.Update(ctx, rule); err != nil {
			err = fmt.Errorf("failed to update prometheus rule: %s", err.Error())
		}
	} else {
		log.Error(err, "failed to get old rule")
		if err = c.Create(ctx, rule); err != nil {
			err = fmt.Errorf("failed to create prometheus rule: %s", err.Error())
		}
	}

	alertsInitialized = true
	return
}

// NewPrometheusRule creates a new PrometheusRule object
func NewPrometheusRule(namespace string) *prometheus.PrometheusRule {
	return &prometheus.PrometheusRule{
		TypeMeta: metav1.TypeMeta{
			APIVersion: prometheus.SchemeGroupVersion.String(),
			Kind:       "PrometheusRule",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ruleName,
			Namespace: namespace,
		},
		Spec: *NewPrometheusRuleSpec(),
	}
}

// NewPrometheusRuleSpec creates a new PrometheusRuleSpec object
func NewPrometheusRuleSpec() *prometheus.PrometheusRuleSpec {
	var rules []prometheus.Rule

	// Add rules for counting backup operator runs by various dimensions
	rules = append(rules, func() (r []prometheus.Rule) {
		for _, by := range []string{"namespace", "storage", "schedule", "state"} {
			r = append(r, prometheus.Rule{
				Record: fmt.Sprintf("%s:%s_run:count", by, metricsNamespace),
				Expr: intstr.FromString(
					fmt.Sprintf("count by (%s) (%s)", by, BackupOperatorRunStatusFullName),
				),
			})
		}
		return
	}()...)

	// Add rules for counting backup operator schedules by namespace and storage
	rules = append(rules, func() (r []prometheus.Rule) {
		for _, by := range []string{"namespace", "storage"} {
			r = append(r, prometheus.Rule{
				Record: fmt.Sprintf("%s:%s_schedule:count", by, metricsNamespace),
				Expr: intstr.FromString(
					fmt.Sprintf("count by (%s) (%s)", by, BackupOperatorScheduleStatusFullName),
				),
			})
		}
		return
	}()...)

	// Add rule for counting backup operator storage
	rules = append(rules, prometheus.Rule{
		Record: fmt.Sprintf("%s_storage:count", metricsNamespace),
		Expr: intstr.FromString(
			fmt.Sprintf("count(%s)", BackupOperatorStorageStatusFullName),
		),
	})

	return &prometheus.PrometheusRuleSpec{
		Groups: []prometheus.RuleGroup{{
			Name:  alertRuleGroup,
			Rules: rules,
		}},
	}
}
