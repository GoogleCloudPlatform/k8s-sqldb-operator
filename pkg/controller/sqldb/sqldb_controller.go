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
	"k8s.io/sqldb/pkg/utils"
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

// TODO (sunilarora): Change defaulting codes to a webhook.
func (r *ReconcileSqlDB) defaultFields(instance *operatorv1alpha1.SqlDB) error {
	var defaulted bool

	if instance.Spec.Version == nil {
		defaultVersion := "latest"
		instance.Spec.Version = &defaultVersion
		defaulted = true
	}

	if instance.Spec.Replicas == nil {
		defaultReplicaNumber := int32(1)
		instance.Spec.Replicas = &defaultReplicaNumber
		defaulted = true
	}

	if instance.Spec.Disk.Type == nil {
		defaultDiskType := operatorv1alpha1.ZonalPersistentDisk
		instance.Spec.Disk.Type = &defaultDiskType
		defaulted = true
	}

	if instance.Spec.Disk.SizeGB == nil {
		defaultDiskSizeGB := int32(1)
		instance.Spec.Disk.SizeGB = &defaultDiskSizeGB
		defaulted = true
	}

	if defaulted {
		return r.Update(context.TODO(), instance)
	}
	return nil
}

// TODO (sunilarora): Change validation codes to a webhook.
func validateFields(instance *operatorv1alpha1.SqlDB) error {
	if instance.Spec.Type != operatorv1alpha1.PostgreSQL {
		return fmt.Errorf(".spec.type must be either %q", operatorv1alpha1.PostgreSQL)
	}
	return nil
}

func getImageName(dbType string) string {
	// For PostgreSQL database.
	if dbType == string(operatorv1alpha1.PostgreSQL) {
		return "postgres"
	}
	// For other databases, return empty string for now.
	return ""
}

// getSVCTemplate returns a Service template.
func getSVCTemplate(instanceName string) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sqldb-" + instanceName + "-svc",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"sqldb": instanceName,
			},
			Ports: []corev1.ServicePort{
				{
					Protocol:   "TCP",
					Port:       5432,
					TargetPort: intstr.FromInt(5432),
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
	// TODO: Figure out what to do with this Service.
	svc := getSVCTemplate(instance.Name)
	// Set SqlDB resource to own the service resource.
	if err := controllerutil.SetControllerReference(instance, svc, r.scheme); err != nil {
		return reconcile.Result{}, err
	}
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
	// TODO (sunilarora): Secret credentials support.
	// TODO (sunilarora): Implement db-client to check for health/status of the DB.
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name + "-statefulset",
			Namespace: instance.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: instance.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"sqldb": instance.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"sqldb": instance.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  strings.ToLower(fmt.Sprintf("%s-db", instance.Spec.Type)),
							Image: fmt.Sprintf("%s:%s", getImageName(string(instance.Spec.Type)), *instance.Spec.Version),
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "sqldb-pvc",
									MountPath: "/sqldb",
								},
							},
						},
					},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				*getPVCTemplate(*instance.Spec.Disk.SizeGB),
			},
			ServiceName: "sqldb-" + instance.Name + "-svc",
		},
	}

	// Set SqlDB resource to own the StatefulSet resource.
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
		instance.Status.Phase = operatorv1alpha1.ServerDeploymentInProgress
		return reconcile.Result{}, r.Update(context.TODO(), instance)
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// The StatefulSet is not ready yet, so reconcile immediately.
	if foundSts.Status.ReadyReplicas != *foundSts.Spec.Replicas {
		return reconcile.Result{}, nil
	}

	// Update status of SqlDB accordingly when the StatefulSet is ready.
	instance.Status.Phase = operatorv1alpha1.ServerReady
	if err = r.Update(context.TODO(), instance); err != nil {
		return reconcile.Result{}, err
	}

	// After starting the PostgreSQL server, handle from-restore deployment
	// if it is not yet performed (.status.phase field != ServerRestored)
	// and .spec.backupName field is specified.
	if instance.Status.Phase != operatorv1alpha1.ServerRestored && instance.Spec.BackupName != nil {
		sqlBackup := &operatorv1alpha1.SqlBackup{}
		sqlBackupName := *instance.Spec.BackupName
		err = r.Get(context.TODO(), types.NamespacedName{Name: sqlBackupName, Namespace: instance.Namespace}, sqlBackup)
		if err != nil && errors.IsNotFound(err) {
			return reconcile.Result{},
				fmt.Errorf("SqlBackup resource named %q does not exist although it is specified by .spec.backupName field", sqlBackupName)
		}
		backupFileName := *sqlBackup.Spec.FileName
		if backupFileName == "" {
			backupFileName = "db.dump"
		}
		var cmd string
		if instance.Spec.Type == operatorv1alpha1.PostgreSQL {
			cmd = fmt.Sprintf("pg_restore -U postgres -d postgres sqldb/%s", backupFileName)
			if err = utils.PerformOperation("postgresql-db", cmd); err != nil {
				return reconcile.Result{}, err
			}
		}
		// Update status of SqlDB accordingly.
		instance.Status.Phase = operatorv1alpha1.ServerRestored
		if err = r.Update(context.TODO(), instance); err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}
