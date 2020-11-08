package metrics

import (
	"context"
	"os"
	"sort"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubectl/pkg/cmd/top"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/metricsutil"
	metricsapi "k8s.io/metrics/pkg/apis/metrics"
	metricsV1beta1api "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"
)

var (
	MeasuredResources = []v1.ResourceName{
		v1.ResourceCPU,
		v1.ResourceMemory,
	}
	NodeColumns     = []string{"NAME", "CPU(cores)", "CPU%", "MEMORY(bytes)", "MEMORY%"}
	PodColumns      = []string{"NAME", "CPU(cores)", "MEMORY(bytes)"}
	NamespaceColumn = "NAMESPACE"
	PodColumn       = "POD"
)

type NodeMetricsValues struct {
	Name       string
	CPUPercent int
	MemPercent int
	CPUCores   int
	MemCores   int
}

func getClients() (*kubernetes.Clientset, *metricsclientset.Clientset, error) {
	kubeConfigFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	matchVersionKubeConfigFlags := cmdutil.NewMatchVersionFlags(kubeConfigFlags)
	f := cmdutil.NewFactory(matchVersionKubeConfigFlags)

	var err error
	config, err := f.ToRESTConfig()
	if err != nil {
		return nil, nil, err
	}
	clientSet, err := f.KubernetesClientSet()
	if err != nil {
		return nil, nil, err
	}
	metricsClient, err := metricsclientset.NewForConfig(config)
	return clientSet, metricsClient, err
}

func GetNodeMetrics() ([]NodeMetricsValues, error) {
	ioStreams := genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}
	o := &top.TopNodeOptions{
		IOStreams: ioStreams,
	}
	clientset, metricsClient, err := getClients()
	if err != nil {
		return nil, err
	}
	o.MetricsClient = metricsClient
	o.NodeClient = clientset.CoreV1()
	o.Printer = metricsutil.NewTopCmdPrinter(o.Out)

	versionedMetrics := &metricsV1beta1api.NodeMetricsList{}
	mc := o.MetricsClient.MetricsV1beta1()
	nm := mc.NodeMetricses()
	versionedMetrics, err = nm.List(context.TODO(), metav1.ListOptions{LabelSelector: labels.Everything().String()})
	if err != nil {
		return nil, err
	}
	metrics := &metricsapi.NodeMetricsList{}
	err = metricsV1beta1api.Convert_v1beta1_NodeMetricsList_To_metrics_NodeMetricsList(versionedMetrics, metrics, nil)
	if err != nil {
		return nil, err
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

	values := []NodeMetricsValues{}
	for _, m := range metrics.Items {
		cpuQuantity := m.Usage[v1.ResourceCPU]
		cpuAvailable := allocatable[m.Name][v1.ResourceCPU]
		cpuFraction := int64(float64(cpuQuantity.MilliValue()) / float64(cpuAvailable.MilliValue()) * 100)
		memQuantity := m.Usage[v1.ResourceMemory]
		memAvailable := allocatable[m.Name][v1.ResourceMemory]
		memFraction := int64(float64(memQuantity.MilliValue()) / float64(memAvailable.MilliValue()) * 100)
		values = append(values, NodeMetricsValues{
			Name:       m.Name,
			CPUPercent: int(cpuFraction),
			MemPercent: int(memFraction),
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
