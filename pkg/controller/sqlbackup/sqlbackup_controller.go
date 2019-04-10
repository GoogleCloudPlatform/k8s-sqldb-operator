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

	infrav1alpha1 "github.com/k8s-sqldb-operator/pkg/apis/infra/v1alpha1"
	"github.com/k8s-sqldb-operator/pkg/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	DefaultUsername = "john"
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
	err = c.Watch(&source.Kind{Type: &infrav1alpha1.SqlBackup{}}, &handler.EnqueueRequestForObject{})
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

func (r *ReconcileSqlBackup) defaultFields(instance *infrav1alpha1.SqlBackup) error {
	if instance.Spec.FileName == nil {
		defaultFileName := "db.dump"
		instance.Spec.FileName = &defaultFileName
		return r.Update(context.TODO(), instance)
	}
	return nil
}

// Reconcile reads that state of the cluster for a SqlBackup object and makes changes based on the state read
// and what is in the SqlBackup.Spec
// +kubebuilder:rbac:groups=infra.example.com,resources=sqlbackups,verbs=get;list;watch;create;update;patch;delete
func (r *ReconcileSqlBackup) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the SqlBackup instance.
	instance := &infrav1alpha1.SqlBackup{}
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
	db := &infrav1alpha1.SqlDB{}
	err = r.Get(context.TODO(), types.NamespacedName{Name: instance.Spec.SqlDBName, Namespace: instance.Namespace}, db)
	if err != nil && errors.IsNotFound(err) {
		return reconcile.Result{}, fmt.Errorf("SqlDB resource named %q does not exist", instance.Spec.SqlDBName)
	}

	// Trigger backup for PostgreSQL database.
	if db.Spec.Type == infrav1alpha1.PostgreSQL {
		fileName := *instance.Spec.FileName
		cmd := fmt.Sprintf("pg_dump -U %s -Fc > sqldb/%s", DefaultUsername, fileName)
		if err = utils.PerformOperation("postgresql-db", instance.Spec.SqlDBName, cmd); err != nil {
			// Update status of SqlBackup after the backup has failed.
			instance.Status.Phase = infrav1alpha1.BackupFailed
			if updateErr := r.Update(context.TODO(), instance); updateErr != nil {
				return reconcile.Result{}, fmt.Errorf("after failing to perform backup, failed to update SqlBackup: %+v", updateErr)
			}
			return reconcile.Result{}, err
		}
	}

	// Update status of SqlBackup after performing the backup successfully.
	instance.Status.Phase = infrav1alpha1.BackupSucceeded
	if updateErr := r.Update(context.TODO(), instance); updateErr != nil {
		return reconcile.Result{}, fmt.Errorf("after successfully performing backup, failed to update SqlBackup: %+v", updateErr)
	}

	return reconcile.Result{}, nil
}
