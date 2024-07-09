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
	"os"
	"strings"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	client "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ruleName           = "backup-operator-rules"
	alertRuleGroup     = "backup-operator.rules"
	runbookURLBasePath = "https://backup-operator.io/metrics"
)

var (
	alertsEnabled   = !strings.Contains("false,no", strings.ToLower(os.Getenv("CREATE_ALERTS")))
	alertsNamespace = os.Getenv("ALERTS_NAMESPACE")
)

func RegisterAlerts() {
	ctx := context.Background()
	log := ctrl.Log.WithName("register-alerts")

	if !alertsEnabled {
		log.Info("alert rules creation is disabled")
		return
	}
	if alertsNamespace == "" {
		namespaceFile := "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
		if namespace, err := os.ReadFile(namespaceFile); err != nil {
			alertsNamespace = string(namespace)
		} else {
			log.Error(err, "alert rules namespace is not defined and could not be obtained from serviceAccount file")
			os.Exit(1)
		}
	}

	c, err := client.New(ctrl.GetConfigOrDie(), client.Options{})
	if err != nil {
		log.Error(err, "failed to create kubernetes client")
		os.Exit(1)
	}

	rule := NewPrometheusRule(alertsNamespace)
	existingRule := &monitoringv1.PrometheusRule{}
	if err = c.Get(ctx, types.NamespacedName{Name: ruleName, Namespace: metricsNamespace}, existingRule); err == nil {
		rule.SetResourceVersion(existingRule.GetResourceVersion())
		if err = c.Update(ctx, rule); err != nil {
			log.Error(err, "failed to update prometheus rule")
			os.Exit(1)
		}
	} else {
		if err = c.Create(ctx, rule); err != nil {
			log.Error(err, "failed to create prometheus rule")
			os.Exit(1)
		}
	}
}

// NewPrometheusRule creates new PrometheusRule(CR) for the operator to have alerts and recording rules
func NewPrometheusRule(namespace string) *monitoringv1.PrometheusRule {
	return &monitoringv1.PrometheusRule{
		TypeMeta: metav1.TypeMeta{
			APIVersion: monitoringv1.SchemeGroupVersion.String(),
			Kind:       "PrometheusRule",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ruleName,
			Namespace: namespace,
		},
		Spec: *NewPrometheusRuleSpec(),
	}
}

// NewPrometheusRuleSpec creates PrometheusRuleSpec for alerts and recording rules
func NewPrometheusRuleSpec() *monitoringv1.PrometheusRuleSpec {
	return &monitoringv1.PrometheusRuleSpec{
		Groups: []monitoringv1.RuleGroup{{
			Name:  alertRuleGroup,
			Rules: []monitoringv1.Rule{},
		}},
	}
}

// // createOperatorUpTotalRecordingRule creates memcached_operator_up_total recording rule
// func createOperatorUpTotalRecordingRule() monitoringv1.Rule {
// 	return monitoringv1.Rule{
// 		Record: operatorUpTotalRecordingRule,
// 		Expr:   intstr.FromString("sum(up{pod=~'memcached-operator-controller-manager-.*'} or vector(0))"),
// 	}
// }

// // createDeploymentSizeUndesiredAlertRule creates MemcachedDeploymentSizeUndesired alert rule
// func createDeploymentSizeUndesiredAlertRule() monitoringv1.Rule {
// 	return monitoringv1.Rule{
// 		Alert: deploymentSizeUndesiredAlert,
// 		Expr:  intstr.FromString("increase(memcached_deployment_size_undesired_count_total[5m]) >= 3"),
// 		Annotations: map[string]string{
// 			"description": "Memcached-sample deployment size was not as desired more than 3 times in the last 5 minutes.",
// 		},
// 		Labels: map[string]string{
// 			"severity":    "warning",
// 			"runbook_url": runbookURLBasePath + "/MemcachedDeploymentSizeUndesired.md",
// 		},
// 	}
// }

// // createOperatorDownAlertRule creates MemcachedOperatorDown alert rule
// func createOperatorDownAlertRule() monitoringv1.Rule {
// 	return monitoringv1.Rule{
// 		Alert: operatorDownAlert,
// 		Expr:  intstr.FromString("memcached_operator_up_total == 0"),
// 		Annotations: map[string]string{
// 			"description": "No running memcached-operator pods were detected in the last 5 min.",
// 		},
// 		For: "5m",
// 		Labels: map[string]string{
// 			"severity":    "critical",
// 			"runbook_url": runbookURLBasePath + "/MemcachedOperatorDown.md",
// 		},
// 	}
// }
