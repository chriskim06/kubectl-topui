package metrics

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
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

type MetricsValues struct {
	Name       string
	CPUPercent float64
	MemPercent float64
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
