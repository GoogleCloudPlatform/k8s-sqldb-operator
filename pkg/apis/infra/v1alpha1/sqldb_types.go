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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DBType string

const (
	// PostgreSQL database.
	PostgreSQL DBType = "PostgreSQL"
)

type DiskType string

const (
	// Zonal standard persistent disk.
	ZonalPersistentDisk DiskType = "ZonalPersistentDisk"
)

type DBPhase string

const (
	// Deployment of database instances is in progress.
	ServerDeploymentInProgress DBPhase = "ServerDeploymentInProgress"

	// Deployment of database instances has completed.
	// The database instances are ready to accept requests.
	ServerReady DBPhase = "ServerReady"

	// Database instances have been restored from a backup file.
	// The database instances are ready to accept requests.
	ServerRestored DBPhase = "ServerRestored"
)

// SqlDBSpec defines the desired state of SqlDB
type SqlDBSpec struct {
	// Sql database type.
	// Currently only support "PostgreSQL" type.
	Type DBType `json:"type"`

	// Version of the database (e.g., "1.5.1", "latest").
	// Default to "latest" if not specified.
	// +optional
	Version *string `json:"version,omitempty"`

	// Number of database instances.
	// Default to 1 if not specified.
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// Details of underlying disk that stores SQL dumps.
	Disk DBDisk `json:"disk"`

	// Name of SqlBackup resource.
	// If specified, it means restoring the database instances loaded with backup data.
	// +optional
	BackupName *string `json:"backupName,omitempty"`
}

type DBDisk struct {
	// Disk type.
	// Currently support only "ZonalPersistentDisk" type for demo purpose.
	// Default to "ZonalPersistentDisk" if not specified.
	// +optional
	Type *DiskType `json:"type,omitempty"`

	// Disk size in GB.
	// Default to 1 if not specified.
	// +optional
	SizeGB *int32 `json:"sizeGB,omitempty"`
}

// SqlDBStatus defines the observed state of SqlDB
type SqlDBStatus struct {
	// Status of deployment of database instances.
	// +optional
	Phase DBPhase `json:"phase,omitempty"`

	// Endpoint exposes the SQLDB instance.
	Endpoint string `json:"endpoint,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SqlDB is the Schema for the sqldbs API
// +k8s:openapi-gen=true
// +kubebuilder:printcolumn:name="Replicas",type="integer",JSONPath=".spec.replicas",description="The number of replicas launched"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase",description="status of the DB"
// +kubebuilder:printcolumn:name="Endpoint",type="string",JSONPath=".status.endpoint",description="Endpoint to access the DB"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type SqlDB struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired behavior of SqlDB.
	// +optional
	Spec SqlDBSpec `json:"spec,omitempty"`

	// Most recently observed status of SqlDB.
	// This data may be out of date by some window of time.
	// Populated by the system.
	// Read-only.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#spec-and-status
	// +optional
	Status SqlDBStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SqlDBList contains a list of SqlDB
type SqlDBList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SqlDB `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SqlDB{}, &SqlDBList{})
}
