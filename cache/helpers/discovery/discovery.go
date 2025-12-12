package discovery

import (
	"context"
	"fmt"
	"os"
	"time"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Discovery struct {
	ns          string
	serviceName string
	kube        *kubernetes.Clientset
}

func NewDiscovery() (*Discovery, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	serviceName, ok := os.LookupEnv("SERVICE_NAME")
	if !ok {
		return nil, fmt.Errorf("failed to lookup env SERVICE_NAME")
	}
	namespace, ok := os.LookupEnv("NAMESPACE")
	if !ok {
		return nil, fmt.Errorf("failed to lookup environment variable NAMESPACE")
	}

	return &Discovery{
		ns:          namespace,
		serviceName: serviceName,
		kube:        clientset,
	}, nil
}

func (d *Discovery) GetParents() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// EndpointSlices for a Service are labeled with kubernetes.io/service-name=<serviceName>
	sel := labels.Set{
		"kubernetes.io/service-name": d.serviceName,
	}.AsSelector().String()

	slices, err := d.kube.DiscoveryV1().EndpointSlices(d.ns).List(ctx, metaV1.ListOptions{
		LabelSelector: sel,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list endpointslices for service %s: %w", d.serviceName, err)
	}

	seen := make(map[string]struct{}, 64)
	parents := make([]string, 0, 16)

	for _, es := range slices.Items {
		for _, ep := range es.Endpoints {
			// If readiness is specified, only include Ready endpoints
			if ep.Conditions.Ready != nil && !*ep.Conditions.Ready {
				continue
			}
			for _, addr := range ep.Addresses {
				if _, ok := seen[addr]; ok {
					continue
				}
				seen[addr] = struct{}{}
				parents = append(parents, addr)
			}
		}
	}

	return parents, nil
}
