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
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/migtools/oadp-cli/cmd/shared"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/vmware-tanzu/velero/pkg/client"
	appsv1 "k8s.io/api/apps/v1"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// SetupOptions holds the options for the setup command
type SetupOptions struct {
	Force bool // Re-run detection even if already configured

	// Internal state
	detectionResult DetectionResult
	kbClient        kbclient.Client
}

// BindFlags binds the flags to the command
func (o *SetupOptions) BindFlags(flags *pflag.FlagSet) {
	flags.BoolVar(&o.Force, "force", false, "Re-run detection even if already configured")
}

// Complete completes the options
func (o *SetupOptions) Complete(args []string, f client.Factory) error {
	// Create Kubernetes client with apps/v1 types for deployment detection
	kbClient, err := shared.NewClientWithScheme(f, shared.ClientOptions{
		IncludeCoreTypes: true,
		Timeout:          10 * time.Second, // Prevent hanging on cluster connection issues
	})
	if err != nil {
		// Check if this is an authentication error
		if strings.Contains(err.Error(), "Unauthorized") {
			return fmt.Errorf("not logged in to cluster. Please run: oc login <cluster-url>")
		}
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	// Add apps/v1 types to the scheme for deployment access
	if err := appsv1.AddToScheme(kbClient.Scheme()); err != nil {
		return fmt.Errorf("failed to add apps/v1 types to scheme: %w", err)
	}

	o.kbClient = kbClient
	return nil
}

// Validate validates the options
func (o *SetupOptions) Validate(c *cobra.Command, args []string, f client.Factory) error {
	// No validation needed for setup command
	return nil
}

// Run executes the setup command
func (o *SetupOptions) Run(c *cobra.Command, f client.Factory) error {
	fmt.Println("Detecting OADP configuration...")
	fmt.Println()

	// Silence usage help on errors during Run (we provide clear error messages)
	c.SilenceUsage = true

	// Check if already configured (unless --force flag set)
	if !o.Force {
		existingConfig, err := shared.ReadVeleroClientConfig()
		if err != nil {
			return fmt.Errorf("failed to read existing config: %w", err)
		}

		// Check if nonadmin field is explicitly set (not nil)
		if existingConfig.NonAdmin != nil {
			fmt.Println("OADP CLI is already configured.")
			fmt.Println()
			o.printCurrentConfig(existingConfig)
			fmt.Println()
			fmt.Println("To reconfigure, run: oc oadp setup --force")
			return nil
		}
	}

	// Run detection
	ctx := context.Background()
	o.detectionResult = detectUserMode(ctx, o.kbClient)

	// Handle detection errors
	if o.detectionResult.Error != nil {
		// Provide specific guidance based on error type
		errMsg := o.detectionResult.Error.Error()
		if strings.Contains(errMsg, "not logged in") || strings.Contains(errMsg, "Unauthorized") {
			fmt.Println("Error: Not logged in to cluster")
			fmt.Println()
			fmt.Println("Please log in to your cluster:")
			fmt.Println("  oc login <cluster-url>")
			return fmt.Errorf("not logged in to cluster")
		} else {
			fmt.Printf("Error: %v\n", o.detectionResult.Error)
			fmt.Println()
			fmt.Println("This could mean:")
			fmt.Println("  - Your cluster is not accessible")
			fmt.Println("  - Your kubeconfig is invalid")
			fmt.Println("  - Network connectivity issues")
			return o.detectionResult.Error
		}
	}

	// Read existing config to preserve fields like default-nabsl
	config, err := shared.ReadVeleroClientConfig()
	if err != nil {
		return fmt.Errorf("failed to read existing config: %w", err)
	}

	// Update config based on detection result
	if o.detectionResult.IsAdmin {
		config.NonAdmin = false
		config.OADPNamespace = o.detectionResult.OADPNamespace
		// Set namespace to OADP namespace for admin mode
		if config.Namespace == "" {
			config.Namespace = o.detectionResult.OADPNamespace
		}
	} else {
		config.NonAdmin = true
		// Don't set OADP namespace for non-admin users
		config.OADPNamespace = ""
	}

	// Write config file
	if err := shared.WriteVeleroClientConfig(config); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Print success message
	o.printSetupSuccess(config)

	return nil
}

// printCurrentConfig prints the current configuration
func (o *SetupOptions) printCurrentConfig(config *shared.ClientConfig) {
	homeDir, _ := os.UserHomeDir()
	configPath := filepath.Join(homeDir, ".config", "velero", "config.json")

	if config.IsNonAdmin() {
		fmt.Println("Current mode: non-admin")
	} else {
		fmt.Println("Current mode: admin")
		if config.OADPNamespace != "" {
			fmt.Printf("OADP namespace: %s\n", config.OADPNamespace)
		}
	}
	fmt.Printf("Configuration file: %s\n", configPath)
}

// printSetupSuccess prints a success message after setup
func (o *SetupOptions) printSetupSuccess(config *shared.ClientConfig) {
	homeDir, _ := os.UserHomeDir()
	configPath := filepath.Join(homeDir, ".config", "velero", "config.json")

	if o.detectionResult.IsAdmin {
		fmt.Printf("✓ Found OADP controller in namespace: %s\n", o.detectionResult.OADPNamespace)
		fmt.Println("✓ Admin mode enabled")
		fmt.Println()
		fmt.Printf("Configuration saved to: %s\n", configPath)
		fmt.Println()
		fmt.Println("You can now use OADP admin commands:")
		fmt.Println("  oc oadp backup create my-backup")
		fmt.Println("  oc oadp restore create my-restore")
	} else {
		fmt.Println("✗ OADP controller deployment not accessible.")
		fmt.Println("✓ Non-admin mode enabled")
		fmt.Println()
		fmt.Printf("Configuration saved to: %s\n", configPath)
		fmt.Println()
		fmt.Println("You can now use OADP non-admin commands:")
		fmt.Println("  oc oadp nonadmin backup create my-backup")
		fmt.Println("  oc oadp nonadmin restore create my-restore")
		fmt.Println()
		fmt.Println("Note: OADP controller deployment not found or you don't have")
		fmt.Println("cluster-wide permissions. Non-admin mode uses namespace-scoped resources.")
	}
}

// NewSetupCommand creates the setup command
func NewSetupCommand(f client.Factory) *cobra.Command {
	o := &SetupOptions{}

	c := &cobra.Command{
		Use:   "setup",
		Short: "Auto-detect and configure admin vs non-admin mode",
		Long: `Auto-detect and configure admin vs non-admin mode.

This command detects whether you have cluster-wide admin permissions and
automatically configures the OADP CLI to use the appropriate mode:

- Admin mode: Full access to OADP resources across all namespaces
- Non-admin mode: Namespace-scoped access using NonAdminBackup resources

The detection works by checking if you can list the OADP controller deployment
across all namespaces. Admin users can see resources cluster-wide, while
non-admin users are limited to their current namespace.

Configuration is saved to: ~/.config/velero/config.json

Examples:
  # Auto-detect and configure OADP CLI
  oc oadp setup

  # Re-run detection (reconfigure)
  oc oadp setup --force`,
		Args: cobra.ExactArgs(0),
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(args, f); err != nil {
				return err
			}
			if err := o.Validate(c, args, f); err != nil {
				return err
			}
			return o.Run(c, f)
		},
	}

	o.BindFlags(c.Flags())

	return c
}
