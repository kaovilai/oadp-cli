/*
Copyright 2025 The OADP CLI Contributors.

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

package shared

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"

	nacv1alpha1 "github.com/migtools/oadp-non-admin/api/v1alpha1"
)

// FindNABSLRequestByNameOrUUID finds a NonAdminBackupStorageLocationRequest by either:
// 1. Direct UUID lookup (if nameOrUUID is the actual request UUID)
// 2. NABSL name lookup (searches through all requests to find one with matching source NABSL name)
//
// This handles the common pattern where users can specify either the NABSL-friendly name
// or the system-generated UUID for approval/rejection operations.
func FindNABSLRequestByNameOrUUID(ctx context.Context, client kbclient.WithWatch, nameOrUUID string, adminNamespace string) (string, error) {

	// UUID lookup
	var testRequest nacv1alpha1.NonAdminBackupStorageLocationRequest
	err := client.Get(ctx, kbclient.ObjectKey{
		Name:      nameOrUUID,
		Namespace: adminNamespace,
	}, &testRequest)
	if err == nil {
		return nameOrUUID, nil
	} else if errors.IsNotFound(err) {
		return "", fmt.Errorf("request for NABSL %q not found", nameOrUUID)
	}

	// Fallback:Match NABSL name to UUID
	var requestList nacv1alpha1.NonAdminBackupStorageLocationRequestList
	err = client.List(ctx, &requestList, kbclient.InNamespace(adminNamespace))
	if err != nil {
		return "", fmt.Errorf("failed to list requests: %w", err)
	}

	for _, request := range requestList.Items {
		if request.Status.SourceNonAdminBSL != nil &&
			request.Status.SourceNonAdminBSL.Name == nameOrUUID {
			return request.Name, nil
		}
	}

	return "", fmt.Errorf("request for NABSL %q not found", nameOrUUID)
}
