/*
Copyright © 2020 Chris Kim

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
package metrics

import (
	"context"
	"errors"
	"fmt"
	"sort"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/kubectl/pkg/cmd/top"
	"k8s.io/kubectl/pkg/metricsutil"
	metricsapi "k8s.io/metrics/pkg/apis/metrics"
	metricsV1beta1api "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	"sigs.k8s.io/yaml"
)

// GetNodeMetrics returns a slice of objects that are meant to be easily
// consumable by the various termui widgets
func (m MetricsClient) GetNodeMetrics(o *top.TopNodeOptions) ([]MetricValue, error) {
	o.MetricsClient = m.m
	o.NodeClient = m.k.CoreV1()
	o.Printer = metricsutil.NewTopCmdPrinter(o.Out)
	if o.SortBy != "" && o.SortBy != "cpu" && o.SortBy != "memory" {
		return nil, errors.New(fmt.Sprintf("invalid sort-by provided: %s", o.SortBy))
	}

	mc := o.MetricsClient.MetricsV1beta1()
	nm := mc.NodeMetricses()

	// handle getting all or with resource name
	versionedMetrics, err := nm.List(context.TODO(), metav1.ListOptions{LabelSelector: labels.Everything().String()})
	if err != nil {
		return nil, err
	}
	metrics := &metricsapi.NodeMetricsList{}
	err = metricsV1beta1api.Convert_v1beta1_NodeMetricsList_To_metrics_NodeMetricsList(versionedMetrics, metrics, nil)
	if err != nil {
		return nil, err
	}

	if o.SortBy != "" {
		sort.Sort(metricsutil.NewNodeMetricsSorter(metrics.Items, o.SortBy))
	}

	nodeList, err := o.NodeClient.Nodes().List(context.TODO(), metav1.ListOptions{
		LabelSelector: labels.Everything().String(),
	})
	if err != nil {
		return nil, err
	}
	var nodes []v1.Node
	nodes = append(nodes, nodeList.Items...)
	allocatable := make(map[string]v1.ResourceList)
	for _, n := range nodes {
		allocatable[n.Name] = n.Status.Allocatable
	}

	values := []MetricValue{}
	for _, m := range metrics.Items {
		cpuQuantity := m.Usage[v1.ResourceCPU]
		cpuAvailable := allocatable[m.Name][v1.ResourceCPU]
		cpuFraction := float64(cpuQuantity.MilliValue()) / float64(cpuAvailable.MilliValue()) * 100
		memQuantity := m.Usage[v1.ResourceMemory]
		memAvailable := allocatable[m.Name][v1.ResourceMemory]
		memFraction := float64(memQuantity.MilliValue()) / float64(memAvailable.MilliValue()) * 100
		values = append(values, MetricValue{
			Name:       m.Name,
			CPUCores:   cpuQuantity,
			CPULimit:   cpuAvailable,
			CPUPercent: cpuFraction,
			MemCores:   memQuantity.Value() / DIVISOR,
			MemLimit:   memAvailable.Value() / DIVISOR,
			MemPercent: memFraction,
		})
	}

	if o.SortBy == "" {
		// Sort the metrics alphabetically
		sort.Slice(values, func(i, j int) bool {
			return values[i].Name < values[j].Name
		})
	}

	return values, nil
}

func (m MetricsClient) GetNode(name string) (string, error) {
	node, err := m.k.CoreV1().Nodes().Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	if !m.showManagedFields {
		node.ManagedFields = nil
	}
	s, err := yaml.Marshal(node)
	if err != nil {
		return "", err
	}
	return string(s), nil
}
