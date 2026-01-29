package k8s

import (
	"context"
	"fmt"
	"os"

	"github.com/jabberwocky238/distributor/store"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const WorkerNamespace = "worker"
const IngressNamespace = "ingress"

var ingressRouteGVR = schema.GroupVersionResource{
	Group:    "traefik.io",
	Version:  "v1alpha1",
	Resource: "ingressroutes",
}

type Client struct {
	clientset     *kubernetes.Clientset
	dynamicClient dynamic.Interface
	domain        string
}

func NewClient() (*Client, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	domain := os.Getenv("DOMAIN")
	if domain == "" {
		return nil, fmt.Errorf("DOMAIN environment variable is required")
	}

	return &Client{
		clientset:     clientset,
		dynamicClient: dynamicClient,
		domain:        domain,
	}, nil
}

func (c *Client) Deploy(worker *store.Worker) error {
	ctx := context.Background()
	serviceName := worker.Name()

	if err := c.deployDeployment(ctx, worker, serviceName); err != nil {
		return err
	}
	if err := c.deployService(ctx, worker, serviceName); err != nil {
		return err
	}
	if err := c.deployIngressRoute(ctx, worker, serviceName); err != nil {
		return err
	}
	return nil
}

func (c *Client) deployDeployment(ctx context.Context, worker *store.Worker, name string) error {
	replicas := int32(1)
	labels := map[string]string{
		"app":       name,
		"worker_id": worker.WorkerID,
		"owner_id":  worker.OwnerID,
	}
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: WorkerNamespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  name,
						Image: worker.Image,
						Ports: []corev1.ContainerPort{{
							ContainerPort: int32(worker.Port),
						}},
					}},
				},
			},
		},
	}

	deploymentsClient := c.clientset.AppsV1().Deployments(WorkerNamespace)
	_, err := deploymentsClient.Get(ctx, name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		_, err = deploymentsClient.Create(ctx, deployment, metav1.CreateOptions{})
	} else if err == nil {
		_, err = deploymentsClient.Update(ctx, deployment, metav1.UpdateOptions{})
	}
	return err
}

func (c *Client) deployService(ctx context.Context, worker *store.Worker, name string) error {
	labels := map[string]string{
		"app":       name,
		"worker_id": worker.WorkerID,
		"owner_id":  worker.OwnerID,
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: WorkerNamespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": name},
			Ports: []corev1.ServicePort{{
				Port:     int32(worker.Port),
				Protocol: corev1.ProtocolTCP,
			}},
		},
	}

	servicesClient := c.clientset.CoreV1().Services(WorkerNamespace)
	_, err := servicesClient.Get(ctx, name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		_, err = servicesClient.Create(ctx, service, metav1.CreateOptions{})
	}
	return err
}

func (c *Client) deployIngressRoute(ctx context.Context, worker *store.Worker, name string) error {
	host := fmt.Sprintf("%s.worker.%s", worker.Name(), c.domain)

	ingressRoute := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "traefik.io/v1alpha1",
			"kind":       "IngressRoute",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": IngressNamespace,
				"labels": map[string]interface{}{
					"worker_id": worker.WorkerID,
					"owner_id":  worker.OwnerID,
				},
			},
			"spec": map[string]interface{}{
				"entryPoints": []interface{}{"websecure"},
				"routes": []interface{}{
					map[string]interface{}{
						"match": fmt.Sprintf("Host(`%s`)", host),
						"kind":  "Rule",
						"services": []interface{}{
							map[string]interface{}{
								"name":      name,
								"namespace": WorkerNamespace,
								"port":      worker.Port,
							},
						},
					},
				},
				"tls": map[string]interface{}{
					"secretName": "ingress-tls",
				},
			},
		},
	}

	client := c.dynamicClient.Resource(ingressRouteGVR).Namespace(IngressNamespace)
	_, err := client.Get(ctx, name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		_, err = client.Create(ctx, ingressRoute, metav1.CreateOptions{})
	} else if err == nil {
		_, err = client.Update(ctx, ingressRoute, metav1.UpdateOptions{})
	}
	return err
}

func (c *Client) Delete(worker *store.Worker) error {
	ctx := context.Background()
	name := worker.PodName()

	c.clientset.AppsV1().Deployments(WorkerNamespace).Delete(ctx, name, metav1.DeleteOptions{})
	c.clientset.CoreV1().Services(WorkerNamespace).Delete(ctx, name, metav1.DeleteOptions{})
	c.deleteIngressRoute(ctx, name)
	return nil
}

func (c *Client) deleteIngressRoute(ctx context.Context, name string) error {
	client := c.dynamicClient.Resource(ingressRouteGVR).Namespace(IngressNamespace)
	return client.Delete(ctx, name, metav1.DeleteOptions{})
}
