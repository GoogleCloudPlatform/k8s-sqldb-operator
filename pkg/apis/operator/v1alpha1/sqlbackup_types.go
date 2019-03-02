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

// SqlBackupSpec defines the desired state of SqlBackup
type SqlBackupSpec struct {
	// Name of SqlDB resource that has its database instances performed backup with.
	SqlDBName string `json:"sqlDBName"`
}

// SqlBackupStatus defines the observed state of SqlBackup
type SqlBackupStatus struct {
	// Backup status.
	// True if backup has succeeded.
	Succeeded bool `json:"succeeded,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SqlBackup is the Schema for the sqlbackups API
// +k8s:openapi-gen=true
type SqlBackup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SqlBackupSpec   `json:"spec,omitempty"`
	Status SqlBackupStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SqlBackupList contains a list of SqlBackup
type SqlBackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SqlBackup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SqlBackup{}, &SqlBackupList{})
}
