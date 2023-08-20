/*
Copyright 2023.

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

package controller

import (
	"context"
	"fmt"

	kapp "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	myapigroupv1alpha1 "github.com/HTTP500/MyAppResource/api/v1alpha1"
)

// MyAppResourceReconciler reconciles a MyAppResource object
type MyAppResourceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=my.api.group,resources=myappresources,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=my.api.group,resources=myappresources/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=my.api.group,resources=myappresources/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MyAppResource object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *MyAppResourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here
	instance := &myapigroupv1alpha1.MyAppResource{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	fmt.Println("=== Redis ===")
	if instance.Spec.SpecRedis.Enabled {
		redisPod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "redis-pod",
				Labels: map[string]string{
					"app": "redis",
				},
				Namespace: instance.Namespace,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "redis",
						Image: "redis:latest",
						Ports: []corev1.ContainerPort{
							{
								ContainerPort: 6379,
							},
						},
					},
				},
			},
		}

		redisService := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "redis-service",
				Namespace: instance.Namespace,
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{
					"app": "redis",
				},
				Ports: []corev1.ServicePort{
					{
						Port: 6379,
					},
				},
			},
		}
		err := r.Create(ctx, redisPod)
		if err != nil {
			fmt.Println(err)
		}
		err = r.Create(ctx, redisService)
		if err != nil {
			fmt.Println(err)
		}
	}

	fmt.Println("=== ReplicaSet ===")
	rp := kapp.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "k8spodinfo",
			Namespace: instance.Namespace,
		},
		Spec: kapp.ReplicaSetSpec{
			Replicas: instance.Spec.SpecReplicaCount,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "myapp",
				},
			},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "k8sjob",
							Image: fmt.Sprintf("%s:%s", instance.Spec.SpecImage.Repository, instance.Spec.SpecImage.Tag),
							Env: []corev1.EnvVar{
								{
									Name:  "PODINFO_CACHE_SERVER",
									Value: "tcp://redis-service:6379",
								},
								{
									Name:  "PODINFO_UI_COLOR",
									Value: instance.Spec.SpecUI.Color,
								},
								{
									Name:  "PODINFO_UI_MESSAGE",
									Value: instance.Spec.SpecUI.Message,
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse(instance.Spec.SpecResource.MemoryLimit),
									corev1.ResourceCPU:    resource.MustParse(instance.Spec.SpecResource.CPURequest),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse(instance.Spec.SpecResource.MemoryLimit),
									corev1.ResourceCPU:    resource.MustParse(instance.Spec.SpecResource.CPURequest),
								},
							},
						},
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "k8sjob",
					Namespace: instance.Namespace,
					Labels: map[string]string{
						"app": "myapp",
					},
				},
			},
		},
	}
	err := r.Create(ctx, &rp)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("=== Service ===")
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-app-service",
			Namespace: instance.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "myapp",
			},
			Ports: []corev1.ServicePort{
				{
					Port:       9898,
					TargetPort: intstr.FromInt(9898), // The port your application inside the pod is listening on.
				},
			},
			Type: corev1.ServiceTypeLoadBalancer,
		},
	}
	err = r.Create(ctx, svc)
	if err != nil {
		fmt.Println(err)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MyAppResourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&myapigroupv1alpha1.MyAppResource{}).
		Complete(r)
}
