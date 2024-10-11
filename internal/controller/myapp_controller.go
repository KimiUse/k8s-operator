/*
Copyright 2024.

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
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/json"
	v1 "myoperator/api/v1"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// MyAppReconciler reconciles a MyApp object
type MyAppReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

// +kubebuilder:rbac:groups=jk.jk.com,resources=myapps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=jk.jk.com,resources=myapps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=jk.jk.com,resources=myapps/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MyApp object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *MyAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	//用于在日志记录器中添加键值对的上下文信息。在这个例子中，"myapp" 是键，req.NamespacedName 是值。req.NamespacedName的值是一个 "命名空间/资源对象名称"的组合
	logger := r.Log.WithValues("myapp", req.NamespacedName)
	logger.Info("Reconciling myapp")

	//判断MyApp对象是否存在
	instance := &v1.MyApp{}
	if err := r.Client.Get(ctx, req.NamespacedName, instance); err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	//fmt.Println(instance)
	logger.Info("app kind: " + instance.Kind + ", app name: " + instance.Name)
	if instance.DeletionTimestamp != nil {
		return reconcile.Result{}, nil
	}

	//
	deployment := &appsv1.Deployment{}
	if err := r.Client.Get(ctx, req.NamespacedName, deployment); err != nil {
		if !errors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
		// 1. 不存在，则创建
		// 1-1. 创建 Deployment
		deployment = NewDeployment(instance)
		if err := r.Client.Create(ctx, deployment); err != nil {
			return ctrl.Result{}, err
		}

		// 1-2. 创建 Service
		svc := NewService(instance)
		if err := r.Client.Create(ctx, svc); err != nil {
			return ctrl.Result{}, err
		}

	} else {
		oldSpec := &v1.MyAppSpec{}
		if err := json.Unmarshal([]byte(instance.Annotations["spec"]), oldSpec); err != nil {
			return ctrl.Result{}, err
		}
		fmt.Println(*oldSpec)
		fmt.Println(instance.Spec)
		// 2. 对比更新
		if !reflect.DeepEqual(instance.Spec, *oldSpec) {
			// 2-1. 更新 Deployment 资源
			newDeployment := NewDeployment(instance)
			currDeployment := &appsv1.Deployment{}
			if err := r.Client.Get(ctx, req.NamespacedName, currDeployment); err != nil {
				return ctrl.Result{}, err
			}
			currDeployment.Spec = newDeployment.Spec
			if err := r.Client.Update(ctx, currDeployment); err != nil {
				return ctrl.Result{}, err
			}

			// 2-2. 更新 Service 资源
			newService := NewService(instance)
			currService := &corev1.Service{}
			if err := r.Client.Get(ctx, req.NamespacedName, currService); err != nil {
				return ctrl.Result{}, err
			}
			currIP := currService.Spec.ClusterIP
			currService.Spec = newService.Spec
			currService.Spec.ClusterIP = currIP
			if err := r.Client.Update(ctx, currService); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	// 3. 关联 Annotations
	data, _ := json.Marshal(instance.Spec)
	if instance.Annotations != nil {
		instance.Annotations["spec"] = string(data)
	} else {
		instance.Annotations = map[string]string{"spec": string(data)}
	}
	if err := r.Client.Update(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func NewDeployment(app *v1.MyApp) *appsv1.Deployment {
	labels := map[string]string{"app": app.Name}
	selector := &metav1.LabelSelector{MatchLabels: labels}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(app, schema.GroupVersionKind{
					Group:   v1.GroupVersion.Group,
					Version: v1.GroupVersion.Version,
					Kind:    app.Kind,
				}),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: app.Spec.Replicas,
			Selector: selector,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            app.Name,
							Image:           app.Spec.Image,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: app.Spec.ContainerPort,
								},
							},
						},
					},
				},
			},
		},
	}
}

func NewService(app *v1.MyApp) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(app, schema.GroupVersionKind{
					Group:   v1.GroupVersion.Group,
					Version: v1.GroupVersion.Version,
					Kind:    app.Kind,
				}),
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeNodePort,
			Ports: []corev1.ServicePort{
				{
					Protocol:   corev1.ProtocolTCP,
					Port:       app.Spec.ServicePort,
					TargetPort: intstr.FromInt(int(app.Spec.ContainerPort)),
				},
			},
			Selector: map[string]string{
				"app": app.Name,
			},
		},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *MyAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.MyApp{}).
		Complete(r)
}
