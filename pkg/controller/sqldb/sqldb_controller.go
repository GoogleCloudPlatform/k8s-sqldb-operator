/*
Copyright 2019 The Kubernetes Authors.

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

package sqldb

import (
	"context"
	"fmt"
	"log"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	operatorv1alpha1 "k8s.io/sqldb/pkg/apis/operator/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Add creates a new SqlDB Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileSqlDB{Client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("sqldb-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to SqlDB
	err = c.Watch(&source.Kind{Type: &operatorv1alpha1.SqlDB{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch a StatefulSet created by SqlDB
	err = c.Watch(&source.Kind{Type: &appsv1.StatefulSet{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &operatorv1alpha1.SqlDB{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileSqlDB{}

// ReconcileSqlDB reconciles a SqlDB object
type ReconcileSqlDB struct {
	client.Client
	scheme *runtime.Scheme
}

func (r *ReconcileSqlDB) defaultFields(instance *operatorv1alpha1.SqlDB) error {
	if instance.Spec.Version == nil {
		defaultVersion := "latest"
		instance.Spec.Version = &defaultVersion
	}

	if instance.Spec.Replicas == nil {
		defaultReplicaNumber := int32(1)
		instance.Spec.Replicas = &defaultReplicaNumber
	}

	if instance.Spec.Disk.Type == nil {
		defaultDiskType := operatorv1alpha1.ZonalPersistentDisk
		instance.Spec.Disk.Type = &defaultDiskType
	}

	if instance.Spec.Disk.SizeGB == nil {
		defaultDiskSizeGB := int32(1)
		instance.Spec.Disk.SizeGB = &defaultDiskSizeGB
	}

	return r.Update(context.TODO(), instance)
}

func validateFields(instance *operatorv1alpha1.SqlDB) error {
	if instance.Spec.Type != operatorv1alpha1.PostgreSQL && instance.Spec.Type != operatorv1alpha1.MySQL {
		return fmt.Errorf(".spec.type must be either %q or %q", operatorv1alpha1.PostgreSQL, operatorv1alpha1.MySQL)
	}
	return nil
}

func getImageName(dbType string) string {
	if dbType == string(operatorv1alpha1.PostgreSQL) {
		return "postgres" // PostgreSQL
	}
	return "mysql" // MySQL
}

// getSVCTemplate returns a Service template.
func getSVCTemplate(instanceName string) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sqldb-svc",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"statefulset": instanceName + "-statefulset",
			},
			ClusterIP: "None",
			Ports: []corev1.ServicePort{
				{
					Protocol:   "TCP",
					Port:       1,
					TargetPort: intstr.FromInt(1),
				},
			},
		},
	}
}

// getPVCTemplate returns a PersistentVolumeClaim template with required disk size in GB.
func getPVCTemplate(diskSizeGB int32) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: "sqldb-pvc",
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(fmt.Sprintf("%dGi", diskSizeGB)),
				},
			},
		},
	}
}

// Reconcile reads that state of the cluster for a SqlDB object and makes changes based on the state read
// and what is in the SqlDB.Spec
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.k8s.io,resources=sqldbs,verbs=get;list;watch;create;update;patch;delete
func (r *ReconcileSqlDB) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the SqlDB instance.
	instance := &operatorv1alpha1.SqlDB{}
	err := r.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if err = r.defaultFields(instance); err != nil {
		return reconcile.Result{}, err
	}
	if err = validateFields(instance); err != nil {
		return reconcile.Result{}, err
	}

	// Create a load-balancer Service if the Service is not yet created.
	svc := getSVCTemplate(instance.Name)
	foundSvc := &corev1.Service{}
	err = r.Get(context.TODO(), types.NamespacedName{Name: svc.Name, Namespace: svc.Namespace}, foundSvc)
	if err != nil && errors.IsNotFound(err) {
		log.Printf("Creating Service %s/%s\n", svc.Namespace, svc.Name)
		err = r.Create(context.TODO(), svc)
		if err != nil {
			return reconcile.Result{}, err
		}
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// Define the desired StatefulSet object.
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name + "-statefulset",
			Namespace: instance.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: instance.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"statefulset": instance.Name + "-statefulset",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"statefulset": instance.Name + "-statefulset"}},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  strings.ToLower(string(instance.Spec.Type)),
							Image: fmt.Sprintf("%s:%s", getImageName(string(instance.Spec.Type)), *instance.Spec.Version),
						},
					},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				*getPVCTemplate(*instance.Spec.Disk.SizeGB),
			},
			ServiceName: "sqldb-svc", // Hard-coded - hidden from user.
		},
	}
	if err := controllerutil.SetControllerReference(instance, sts, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Create a StatefulSet if the StatefulSet is not yet created.
	foundSts := &appsv1.StatefulSet{}
	err = r.Get(context.TODO(), types.NamespacedName{Name: sts.Name, Namespace: sts.Namespace}, foundSts)
	if err != nil && errors.IsNotFound(err) {
		log.Printf("Creating StatefulSet %s/%s\n", sts.Namespace, sts.Name)
		err = r.Create(context.TODO(), sts)
		if err != nil {
			return reconcile.Result{}, err
		}
	} else if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}
