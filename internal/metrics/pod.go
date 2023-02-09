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
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/kubectl/pkg/cmd/top"
	"k8s.io/kubectl/pkg/metricsutil"
	metricsapi "k8s.io/metrics/pkg/apis/metrics"
	metricsv1beta1api "k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

const metricsCreationDelay = 2 * time.Minute

type resourceLimits struct {
	CPU int64
	Mem int64
}

// GetPodMetrics returns a slice of objects that are meant to be easily
// consumable by the various termui widgets
func (m MetricsClient) GetPodMetrics(o *top.TopPodOptions) ([]MetricsValues, error) {
	ns, err := getNamespace(m.flags)
	if err != nil {
		return nil, err
	}
	if ns == "" {
		ns = metav1.NamespaceAll
	}
	o.MetricsClient = m.m
	o.PodClient = m.k.CoreV1()
	o.Printer = metricsutil.NewTopCmdPrinter(o.Out)

	versionedMetrics := &metricsv1beta1api.PodMetricsList{}
	mc := o.MetricsClient.MetricsV1beta1()
	pm := mc.PodMetricses(ns)

	selector := labels.Everything()
	if len(o.Selector) > 0 {
		selector, err = labels.Parse(o.Selector)
		if err != nil {
			return nil, err
		}
	}

	// handle getting all or with resource name
	versionedMetrics, err = pm.List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return nil, err
	}
	metrics := &metricsapi.PodMetricsList{}
	err = metricsv1beta1api.Convert_v1beta1_PodMetricsList_To_metrics_PodMetricsList(versionedMetrics, metrics, nil)
	if err != nil {
		return nil, err
	}

	if len(metrics.Items) == 0 {
		// If the API server query is successful but all the pods are newly created,
		// the metrics are probably not ready yet, so we return the error here in the first place.
		e := verifyEmptyMetrics(*o, nil)
		if e != nil {
			return nil, e
		}

		// if we had no errors, be sure we output something.
		if o.AllNamespaces {
			return nil, fmt.Errorf("No resources found\n")
		} else {
			return nil, fmt.Errorf("No resources found in %s namespace.\n", o.Namespace)
		}
	}

	// maybe loop through containers and sum the cpu/mem limits to calculate percentages
	podList, err := o.PodClient.Pods(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var pods []v1.Pod
	pods = append(pods, podList.Items...)
	podMapping := map[string]v1.Pod{}
	for _, pod := range pods {
		podMapping[pod.Name] = pod
	}
	limits := getPodResourceLimits(pods)

	values := []MetricsValues{}
	for _, item := range metrics.Items {
		name := item.Name
		podMetrics := getPodMetrics(&item)
		cpuQuantity := podMetrics[v1.ResourceCPU]
		cpuAvailable := limits[name].CPU
		cpuFraction := float64(cpuQuantity.MilliValue()) / float64(cpuAvailable) * 100
		memQuantity := podMetrics[v1.ResourceMemory]
		memAvailable := limits[name].Mem
		memFraction := float64(memQuantity.MilliValue()) / float64(memAvailable) * 100
		ready, total, restarts := containerStatuses(podMapping[name].Status)
		values = append(values, MetricsValues{
			Name:       name,
			CPUPercent: cpuFraction,
			MemPercent: memFraction,
			CPUCores:   int(cpuQuantity.MilliValue()),
			MemCores:   int(memQuantity.Value() / (1024 * 1024)),
			CPULimit:   cpuAvailable,
			MemLimit:   memAvailable,
			Namespace:  podMapping[name].Namespace,
			Status:     string(podMapping[name].Status.Phase),
			Age:        translateTimestampSince(podMapping[name].CreationTimestamp),
			Restarts:   restarts,
			Ready:      ready,
			Total:      total,
		})
	}

	// Sort the metrics results somehow
	sort.Slice(values, func(i, j int) bool {
		return values[i].Name < values[j].Name
	})

	return values, nil
}

func containerStatuses(stats v1.PodStatus) (int, int, int) {
	var ready, restarts int
	for _, stat := range stats.ContainerStatuses {
		restarts += int(stat.RestartCount)
		if stat.Ready {
			ready++
		}
	}
	for _, stat := range stats.InitContainerStatuses {
		restarts += int(stat.RestartCount)
	}
	return ready, len(stats.ContainerStatuses), restarts
}

func translateTimestampSince(timestamp metav1.Time) string {
	if timestamp.IsZero() {
		return "<unknown>"
	}

	return duration.HumanDuration(time.Since(timestamp.Time))
}

func verifyEmptyMetrics(o top.TopPodOptions, selector labels.Selector) error {
	if len(o.ResourceName) > 0 {
		pod, err := o.PodClient.Pods(o.Namespace).Get(context.TODO(), o.ResourceName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if err := checkPodAge(pod); err != nil {
			return err
		}
	} else {
		opts := metav1.ListOptions{}
		if selector != nil {
			opts.LabelSelector = selector.String()
		}
		pods, err := o.PodClient.Pods(o.Namespace).List(context.TODO(), opts)
		if err != nil {
			return err
		}
		if len(pods.Items) == 0 {
			return nil
		}
		for _, pod := range pods.Items {
			if err := checkPodAge(&pod); err != nil {
				return err
			}
		}
	}
	return errors.New("metrics not available yet")
}

func checkPodAge(pod *v1.Pod) error {
	age := time.Since(pod.CreationTimestamp.Time)
	if age > metricsCreationDelay {
		message := fmt.Sprintf("Metrics not available for pod %s/%s, age: %s", pod.Namespace, pod.Name, age.String())
		return errors.New(message)
	} else {
		return nil
	}
}

func getPodMetrics(m *metricsapi.PodMetrics) v1.ResourceList {
	podMetrics := make(v1.ResourceList)
	for _, res := range metricsutil.MeasuredResources {
		podMetrics[res], _ = resource.ParseQuantity("0")
	}

	for _, c := range m.Containers {
		for _, res := range metricsutil.MeasuredResources {
			quantity := podMetrics[res]
			quantity.Add(c.Usage[res])
			podMetrics[res] = quantity
		}
	}
	return podMetrics
}

func getPodResourceLimits(pods []v1.Pod) map[string]resourceLimits {
	limits := map[string]resourceLimits{}
	for _, pod := range pods {
		var cpuLimit, memLimit int64
		for _, container := range pod.Spec.Containers {
			if len(container.Resources.Limits) != 0 {
				cpuLimit += container.Resources.Limits.Cpu().MilliValue()
				memLimit += container.Resources.Limits.Memory().MilliValue()
			} else {
				cpuLimit += container.Resources.Requests.Cpu().MilliValue()
				memLimit += container.Resources.Requests.Memory().MilliValue()
			}
		}
		limits[pod.Name] = resourceLimits{
			CPU: cpuLimit,
			Mem: memLimit,
		}
	}
	return limits
}
