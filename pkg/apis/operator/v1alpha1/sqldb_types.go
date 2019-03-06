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

	// MySQL database.
	MySQL DBType = "MySQL"
)

type DiskType string

const (
	// Zonal standard persistent disk.
	ZonalPersistentDisk DiskType = "ZonalPersistentDisk"
)

// SqlDBSpec defines the desired state of SqlDB
type SqlDBSpec struct {
	// Sql database type.
	// Currently support "PostgreSQL" and "MySQL" types.
	Type DBType `json:"type"`

	// Version of the database (e.g., "1.5.1", "latest").
	// Default to "latest" if not specified.
	Version *string `json:"version,omitempty"`

	// Number of database instances.
	// Default to 1 if not specified.
	Replicas *int32 `json:"replicas,omitempty"`

	// Name of RelationalBackup resource.
	// If specified, it means creating the database instances loaded with backup data.
	BackupName *string `json:"backupName,omitempty"`

	// Details of underlying disk that stores SQL dumps.
	Disk DBDisk `json:"disk"`
}

type DBDisk struct {
	// Disk type.
	// Currently support only "ZonalPersistentDisk" type for demo purpose.
	// Default to "ZonalPersistentDisk" if not specified.
	Type *DiskType `json:"type,omitempty"`

	// Disk size in GB.
	// Default to 10 if not specified.
	SizeGB *int32 `json:"sizeGB,omitempty"`
}

// SqlDBStatus defines the observed state of SqlDB
type SqlDBStatus struct {
	// Status of deployment of database instances.
	// True if the deployment has completed successfully.
	Ready bool `json:"succeeded,omitempty"`

	// Endpoints of database instances.
	// The number of endpoints should be equal to the number of database instances.
	Endpoints []string `json:"endpoints,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SqlDB is the Schema for the sqldbs API
// +k8s:openapi-gen=true
type SqlDB struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SqlDBSpec   `json:"spec,omitempty"`
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
