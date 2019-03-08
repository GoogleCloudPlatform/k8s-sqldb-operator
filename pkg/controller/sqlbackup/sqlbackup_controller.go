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

package sqlbackup

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	operatorv1alpha1 "k8s.io/sqldb/pkg/apis/operator/v1alpha1"
	"k8s.io/sqldb/pkg/utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Add creates a new SqlBackup Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileSqlBackup{Client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("sqlbackup-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to SqlBackup
	err = c.Watch(&source.Kind{Type: &operatorv1alpha1.SqlBackup{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileSqlBackup{}

// ReconcileSqlBackup reconciles a SqlBackup object
type ReconcileSqlBackup struct {
	client.Client
	scheme *runtime.Scheme
}

// TODO: Change defaulting codes to a webhook.
func (r *ReconcileSqlBackup) defaultFields(instance *operatorv1alpha1.SqlBackup) error {
	if instance.Spec.FileName == nil {
		defaultFileName := "db.dump"
		instance.Spec.FileName = &defaultFileName
		return r.Update(context.TODO(), instance)
	}
	return nil
}

// Reconcile reads that state of the cluster for a SqlBackup object and makes changes based on the state read
// and what is in the SqlBackup.Spec
// +kubebuilder:rbac:groups=operator.k8s.io,resources=sqlbackups,verbs=get;list;watch;create;update;patch;delete
func (r *ReconcileSqlBackup) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the SqlBackup instance.
	instance := &operatorv1alpha1.SqlBackup{}
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

	// Check existence of SqlDB resource.
	db := &operatorv1alpha1.SqlDB{}
	err = r.Get(context.TODO(), types.NamespacedName{Name: instance.Spec.SqlDBName, Namespace: instance.Namespace}, db)
	if err != nil && errors.IsNotFound(err) {
		return reconcile.Result{}, fmt.Errorf("SqlDB resource named %q does not exist", instance.Spec.SqlDBName)
	}

	// Trigger backup for PostgreSQL database.
	if db.Spec.Type == operatorv1alpha1.PostgreSQL {
		cmd := fmt.Sprintf("pg_dump -h localhost -U postgres -Fc > sqldb/%s", *instance.Spec.FileName)
		if err = utils.PerformOperation("postgresql-db", cmd); err != nil {
			return reconcile.Result{}, err
		}
	}
	// Implementation for MySQL database is left as exercise.

	return reconcile.Result{}, nil
}
