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

package nabsl

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/migtools/oadp-cli/cmd/shared"
	nacv1alpha1 "github.com/migtools/oadp-non-admin/api/v1alpha1"
	"github.com/vmware-tanzu/velero/pkg/client"
	"github.com/vmware-tanzu/velero/pkg/cmd"
)

func NewDescribeCommand(f client.Factory) *cobra.Command {
	o := NewDescribeOptions()

	c := &cobra.Command{
		Use:   "describe NAME",
		Short: "Describe a non-admin backup storage location request",
		Args:  cobra.ExactArgs(1),
		Run: func(c *cobra.Command, args []string) {
			cmd.CheckError(o.Complete(args, f))
			cmd.CheckError(o.Validate(c, args, f))
			cmd.CheckError(o.Run(c, f))
		},
		Example: `  # Describe a request by NABSL name
  oc oadp nabsl-request describe my-bsl-request

  # Describe a request by UUID
  oc oadp nabsl-request describe nacuser01-my-bsl-96dfa8b7-3f6f-4c8d-a168-8527b00fbed8`,
	}

	return c
}

type DescribeOptions struct {
	UUID_Name string
	client    kbclient.WithWatch
}

func NewDescribeOptions() *DescribeOptions {
	return &DescribeOptions{}
}

func (o *DescribeOptions) Complete(args []string, f client.Factory) error {
	o.UUID_Name = args[0]

	client, err := shared.NewClientWithScheme(f, shared.ClientOptions{
		IncludeVeleroTypes:   true,
		IncludeNonAdminTypes: true,
	})
	if err != nil {
		return err
	}

	o.client = client
	return nil
}

func (o *DescribeOptions) Validate(c *cobra.Command, args []string, f client.Factory) error {
	return nil
}

func (o *DescribeOptions) Run(c *cobra.Command, f client.Factory) error {
	// Get the admin namespace (from client config) where requests are stored
	adminNS := f.Namespace()

	// Get the request from openshift-adp namespace using the UUID
	var request nacv1alpha1.NonAdminBackupStorageLocationRequest
	requestName, err := shared.FindNABSLRequestByNameOrUUID(context.Background(), o.client, o.UUID_Name, adminNS)
	if err != nil {
		return err
	}

	err = o.client.Get(context.Background(), kbclient.ObjectKey{
		Name:      requestName,
		Namespace: adminNS,
	}, &request)

	if err != nil {
		return fmt.Errorf("failed to get request for %q: %w", requestName, err)
	}

	return describeRequest(&request)
}

func describeRequest(request *nacv1alpha1.NonAdminBackupStorageLocationRequest) error {
	// Name and Namespace
	fmt.Printf("Name:         %s\n", request.Name)
	fmt.Printf("Namespace:    %s\n", request.Namespace)

	// Labels
	shared.PrintLabelsOrAnnotations(os.Stdout, "Labels:       ", request.Labels)

	// Annotations
	shared.PrintLabelsOrAnnotations(os.Stdout, "Annotations:  ", request.Annotations)

	fmt.Printf("\n")

	// Phase (with color)
	fmt.Printf("Phase:  %s\n", shared.ColorizePhase(string(request.Status.Phase)))

	fmt.Printf("\n")

	// Approval Decision
	if request.Spec.ApprovalDecision != "" {
		fmt.Printf("Approval Decision:  %s\n", request.Spec.ApprovalDecision)
		fmt.Printf("\n")
	}

	// Requested NonAdminBackupStorageLocation
	if request.Status.SourceNonAdminBSL != nil {
		source := request.Status.SourceNonAdminBSL
		fmt.Printf("Requested NonAdminBackupStorageLocation:\n")
		fmt.Printf("  Name:       %s\n", source.Name)
		fmt.Printf("  Namespace:  %s\n", source.Namespace)

		if source.NACUUID != "" {
			fmt.Printf("  NACUUID:    %s\n", source.NACUUID)
		}

		fmt.Printf("\n")

		// Requested BackupStorageLocation Spec
		if source.RequestedSpec != nil {
			spec := source.RequestedSpec
			fmt.Printf("Requested BackupStorageLocation Spec:\n")
			fmt.Printf("  Provider:                  %s\n", spec.Provider)
			fmt.Printf("  Object Storage Bucket:     %s\n", spec.ObjectStorage.Bucket)

			if spec.ObjectStorage.Prefix != "" {
				fmt.Printf("  Prefix:                    %s\n", spec.ObjectStorage.Prefix)
			}

			if len(spec.Config) > 0 {
				fmt.Printf("  Config:\n")
				configKeys := make([]string, 0, len(spec.Config))
				for k := range spec.Config {
					configKeys = append(configKeys, k)
				}
				sort.Strings(configKeys)
				for _, k := range configKeys {
					fmt.Printf("    %s: %s\n", k, spec.Config[k])
				}
			}

			if spec.AccessMode != "" {
				fmt.Printf("  Access Mode:               %s\n", spec.AccessMode)
			}

			if spec.BackupSyncPeriod != nil {
				fmt.Printf("  Backup Sync Period:        %s\n", spec.BackupSyncPeriod.String())
			}

			if spec.ValidationFrequency != nil {
				fmt.Printf("  Validation Frequency:      %s\n", spec.ValidationFrequency.String())
			}

			fmt.Printf("\n")
		}
	}

	// Creation Timestamp
	fmt.Printf("Creation Timestamp:  %s\n", request.CreationTimestamp.Time.Format("2006-01-02 15:04:05 -0700 MST"))

	return nil
}
