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

package setup

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// DetectionResult holds the result of detecting user mode and OADP installation
type DetectionResult struct {
	IsAdmin       bool
	OADPNamespace string
	Error         error
}

// detectUserMode detects whether the user has admin permissions and finds the OADP namespace.
// It does this by attempting to list deployments across all namespaces and looking for
// "openshift-adp-controller-manager". Admin users can see resources across all namespaces,
// while non-admin users cannot.
func detectUserMode(ctx context.Context, client kbclient.Client) DetectionResult {
	deployments := &appsv1.DeploymentList{}

	// Attempt to list all deployments across all namespaces.
	// This will fail with permission denied for non-admin users.
	err := client.List(ctx, deployments)

	if err != nil {
		// Check if not logged in - this is a fatal error
		if errors.IsUnauthorized(err) {
			return DetectionResult{Error: fmt.Errorf("not logged in to cluster: %w", err)}
		}
		// Check if permission denied - indicates non-admin user
		if errors.IsForbidden(err) {
			return DetectionResult{IsAdmin: false}
		}
		// Other errors (cluster unreachable, etc.)
		return DetectionResult{Error: fmt.Errorf("failed to query cluster: %w", err)}
	}

	// Filter deployments to find OADP controller
	for _, deployment := range deployments.Items {
		if deployment.Name == "openshift-adp-controller-manager" {
			// Found OADP controller - user is admin
			return DetectionResult{
				IsAdmin:       true,
				OADPNamespace: deployment.Namespace,
			}
		}
	}

	// OADP not installed - default to non-admin
	return DetectionResult{IsAdmin: false}
}
