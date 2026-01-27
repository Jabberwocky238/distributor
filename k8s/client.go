package k8s

import (
	"context"
	"fmt"

	"github.com/jabberwocky238/distributor/store"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const WorkerNamespace = "worker"

type Client struct {
	clientset *kubernetes.Clientset
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

	return &Client{clientset: clientset}, nil
}

func (c *Client) Deploy(worker *store.Worker) error {
	ctx := context.Background()
	serviceName := domainToServiceName(worker.DomainPrefix())

	if err := c.deployDeployment(ctx, worker, serviceName); err != nil {
		return err
	}
	if err := c.deployService(ctx, worker, serviceName); err != nil {
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

func (c *Client) Delete(worker *store.Worker) error {
	ctx := context.Background()
	name := domainToServiceName(worker.DomainPrefix())

	c.clientset.AppsV1().Deployments(WorkerNamespace).Delete(ctx, name, metav1.DeleteOptions{})
	c.clientset.CoreV1().Services(WorkerNamespace).Delete(ctx, name, metav1.DeleteOptions{})
	return nil
}

func domainToServiceName(domain string) string {
	result := make([]byte, 0, len(domain))
	for i := 0; i < len(domain); i++ {
		if domain[i] == '.' {
			result = append(result, '-')
		} else {
			result = append(result, domain[i])
		}
	}
	return string(result)
}
