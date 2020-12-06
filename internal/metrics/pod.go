package metrics

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/top"
	"k8s.io/kubectl/pkg/metricsutil"
	metricsapi "k8s.io/metrics/pkg/apis/metrics"
	metricsv1beta1api "k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

type resourceLimits struct {
	CPU int64
	Mem int64
}

func GetPodMetrics(flags *genericclioptions.ConfigFlags) ([]MetricsValues, error) {
	ioStreams := genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}
	o := &top.TopPodOptions{
		IOStreams: ioStreams,
	}
	clientset, metricsClient, ns, err := getClientsAndNamespace(flags)
	if err != nil {
		return nil, err
	}
	if ns == "" {
		ns = metav1.NamespaceAll
	}
	o.MetricsClient = metricsClient
	o.PodClient = clientset.CoreV1()
	o.Printer = metricsutil.NewTopCmdPrinter(o.Out)

	versionedMetrics := &metricsv1beta1api.PodMetricsList{}
	mc := o.MetricsClient.MetricsV1beta1()
	pm := mc.PodMetricses(ns)

	// handle getting all or with resource name
	versionedMetrics, err = pm.List(context.TODO(), metav1.ListOptions{LabelSelector: labels.Everything().String()})
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
	limits := getPodResourceLimits(pods)

	values := []MetricsValues{}
	for _, item := range metrics.Items {
		// check if printing containers
		podMetrics := getPodMetrics(&item)
		cpuQuantity := podMetrics[v1.ResourceCPU]
		cpuAvailable := limits[item.Name].CPU
		cpuFraction := float64(cpuQuantity.MilliValue()) / float64(cpuAvailable) * 100
		memQuantity := podMetrics[v1.ResourceMemory]
		memAvailable := limits[item.Name].Mem
		memFraction := float64(memQuantity.MilliValue()) / float64(memAvailable) * 100
		values = append(values, MetricsValues{
			Name:       item.Name,
			CPUPercent: cpuFraction,
			MemPercent: memFraction,
			CPUCores:   int(cpuQuantity.MilliValue()),
			MemCores:   int(memQuantity.Value()),
		})
	}

	// Sort the metrics results somehow
	sort.Slice(values, func(i, j int) bool {
		return values[i].Name < values[j].Name
	})

	return values, nil
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
		pods, err := o.PodClient.Pods(o.Namespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: selector.String(),
		})
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

const metricsCreationDelay = 2 * time.Minute

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
	for _, res := range MeasuredResources {
		podMetrics[res], _ = resource.ParseQuantity("0")
	}

	for _, c := range m.Containers {
		for _, res := range MeasuredResources {
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
