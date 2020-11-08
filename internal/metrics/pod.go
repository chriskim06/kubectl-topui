package metrics

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/top"
	"k8s.io/kubectl/pkg/metricsutil"
	metricsapi "k8s.io/metrics/pkg/apis/metrics"
	metricsv1beta1api "k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

func GetPodMetrics(namespace string) ([]MetricsValues, error) {
	ioStreams := genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}
	o := &top.TopPodOptions{
		IOStreams: ioStreams,
	}
	clientset, metricsClient, err := getClients()
	if err != nil {
		return nil, err
	}
	o.MetricsClient = metricsClient
	o.PodClient = clientset.CoreV1()
	o.Printer = metricsutil.NewTopCmdPrinter(o.Out)

	ns := metav1.NamespaceAll
	if namespace != "" {
		ns = namespace
	}
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

	return nil, nil
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
