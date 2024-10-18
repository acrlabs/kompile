package komputil

import (
	"context"
	"fmt"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func CreateAndWaitForPod(name, image string) (string, error) {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	namespace := os.Getenv("POD_NAMESPACE")

	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    namespace,
			GenerateName: name + "-",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  name,
					Image: image,
					Ports: []corev1.ContainerPort{
						{ContainerPort: 8080},
					},
				},
			},
		},
	}

	createdPod, err := clientset.CoreV1().Pods(namespace).Create(
		context.Background(),
		&pod,
		metav1.CreateOptions{},
	)
	if err != nil {
		return "", fmt.Errorf("could not create pod: %w", err)
	}

	for {
		foundPod, err := clientset.CoreV1().Pods(namespace).Get(
			context.Background(),
			createdPod.ObjectMeta.Name,
			metav1.GetOptions{},
		)
		if err != nil {
			return "", fmt.Errorf("could not fetch pod: %w", err)
		}

		if foundPod.Status.Phase == corev1.PodRunning {
			podURL := fmt.Sprintf("http://%s:8080", foundPod.Status.PodIP)
			return podURL, nil
		}
		time.Sleep(time.Second)
	}
}
