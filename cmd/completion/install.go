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

package completion

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/sync/errgroup"
)

const (
	shellBash = "bash"
	shellZsh  = "zsh"
	shellFish = "fish"
)

type shellConfig struct {
	completionSubdir string
	rcFile           string
	filePrefix       string
	fileExtension    string
	completionCheck  func(content string) bool
}

var shellConfigs = map[string]shellConfig{
	shellBash: {
		completionSubdir: filepath.Join(".local", "share", "bash-completion", "completions"),
		rcFile:           ".bashrc",
		filePrefix:       "",
		fileExtension:    "",
		completionCheck: func(content string) bool {
			return strings.Contains(content, "bash_completion")
		},
	},
	shellZsh: {
		completionSubdir: filepath.Join(".zsh", "completions"),
		rcFile:           ".zshrc",
		filePrefix:       "_",
		fileExtension:    "",
		completionCheck: func(content string) bool {
			return strings.Contains(content, "fpath=") && strings.Contains(content, "compinit")
		},
	},
	shellFish: {
		completionSubdir: filepath.Join(".config", "fish", "completions"),
		rcFile:           "",
		filePrefix:       "",
		fileExtension:    ".fish",
		completionCheck: func(content string) bool {
			return true // fish auto-loads, no setup needed
		},
	},
}

type InstallOptions struct {
	Shell string
	Force bool // Force regeneration of completion files (helpful after updates to OADP CLI)
}

// BindFlags binds flags to the command
func (o *InstallOptions) BindFlags(flags *pflag.FlagSet) {
	flags.StringVar(&o.Shell, "shell", "", "Shell to install completions for (bash, zsh, fish)")
	flags.BoolVar(&o.Force, "force", false, "Reinstall completions even if already installed")
}

// Complete completes the options
func (o *InstallOptions) Complete(args []string) error {
	// Auto-detect shell if not specified
	if o.Shell == "" {
		shell := os.Getenv("SHELL")
		if shell != "" {
			o.Shell = filepath.Base(shell)
		}
	}
	return nil
}

// Validate validates the options
func (o *InstallOptions) Validate() error {
	if _, ok := shellConfigs[o.Shell]; !ok {
		return fmt.Errorf("unsupported shell: %s (supported: %s, %s, %s)", o.Shell, shellBash, shellZsh, shellFish)
	}
	return nil
}

// Run executes the install command
func (o *InstallOptions) Run(c *cobra.Command) error {
	//c.SilenceUsage = true

	fmt.Printf("Installing completions for %s...\n", o.Shell)

	// Check for bash-completion if shell is bash
	if o.Shell == shellBash {
		if !o.isBashCompletionInstalled() {
			return o.printBashCompletionError()
		}
	}

	// Check if already installed (unless --force)
	if !o.Force && o.isAlreadyInstalled() {
		fmt.Println("✓ Completions files are already installed")
		fmt.Println("Use --force to reinstall")
		fmt.Println()

		// Still print setup instructions in case user hasn't configured their shell
		o.printSetupInstructions()
		return nil
	}

	// Create completion directory
	dir := o.getCompletionDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create completion directory: %w", err)
	}

	// Generate completion files
	if err := o.generateCompletionFiles(dir); err != nil {
		return fmt.Errorf("failed to generate completion files: %w", err)
	}

	// Print success message with setup instructions
	o.printSuccessMessage()
	return nil
}

// isBashCompletionInstalled checks if bash-completion is available
func (o *InstallOptions) isBashCompletionInstalled() bool {
	// Check common bash-completion paths
	paths := []string{
		"/opt/homebrew/etc/bash_completion",
		"/usr/local/etc/bash_completion",
		"/etc/bash_completion",
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}

	// Try brew --prefix with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "brew", "--prefix")
	if output, err := cmd.Output(); err == nil {
		brewPrefix := strings.TrimSpace(string(output))
		path := filepath.Join(brewPrefix, "etc", "bash_completion")
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}

	return false
}

// printBashCompletionError prints error message for missing bash-completion
func (o *InstallOptions) printBashCompletionError() error {
	fmt.Println("Error: bash-completion package is required for bash shell completions.")
	fmt.Println()
	fmt.Println("To install:")
	fmt.Println("  macOS:       brew install bash-completion")
	fmt.Println("  Ubuntu:      sudo apt-get install bash-completion")
	fmt.Println("  RHEL/Fedora: sudo dnf install bash-completion")
	fmt.Println()
	fmt.Println("After installing, run: kubectl-oadp completion install")
	return fmt.Errorf("bash-completion not installed")
}

// isAlreadyInstalled checks if completions are already installed
func (o *InstallOptions) isAlreadyInstalled() bool {
	dir := o.getCompletionDir()
	config := shellConfigs[o.Shell]

	// Build file names using shell config
	ocFile := filepath.Join(dir, config.filePrefix+"oc"+config.fileExtension)
	oadpFile := filepath.Join(dir, config.filePrefix+"kubectl-oadp"+config.fileExtension)

	// Check if both completion files exist
	_, ocErr := os.Stat(ocFile)
	_, oadpErr := os.Stat(oadpFile)
	return ocErr == nil && oadpErr == nil
}

// getCompletionDir returns the completion directory for the shell
func (o *InstallOptions) getCompletionDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to HOME env var if UserHomeDir fails
		home = os.Getenv("HOME")
	}
	config := shellConfigs[o.Shell]
	return filepath.Join(home, config.completionSubdir)
}

// getRCFile returns the RC file path for the shell
func (o *InstallOptions) getRCFile() string {
	config := shellConfigs[o.Shell]
	if config.rcFile == "" {
		return ""
	}
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to HOME env var if UserHomeDir fails
		home = os.Getenv("HOME")
	}
	return filepath.Join(home, config.rcFile)
}

// generateCompletionFiles generates completion files for oc and kubectl-oadp
func (o *InstallOptions) generateCompletionFiles(dir string) error {
	config := shellConfigs[o.Shell]

	ocFile := filepath.Join(dir, config.filePrefix+"oc"+config.fileExtension)
	oadpFile := filepath.Join(dir, config.filePrefix+"kubectl-oadp"+config.fileExtension)

	// Generate completions in parallel
	var g errgroup.Group
	g.Go(func() error {
		return o.generateCompletion("oc", ocFile)
	})
	g.Go(func() error {
		return o.generateCompletion("kubectl-oadp", oadpFile)
	})

	if err := g.Wait(); err != nil {
		return fmt.Errorf("failed to generate completion files: %w", err)
	}

	fmt.Printf("✓ Generated completion files in %s\n", dir)
	return nil
}

// generateCompletion generates completion for a command
func (o *InstallOptions) generateCompletion(command, outputFile string) error {
	fmt.Printf("  Generating %s completion...\n", command)

	// Use timeout to prevent indefinite hangs
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, "completion", o.Shell)

	// Capture both stdout and stderr for debugging
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run '%s completion %s': %w\nOutput: %s", command, o.Shell, err, string(output))
	}

	return os.WriteFile(outputFile, output, 0644)
}

// printSuccessMessage prints success message with setup instructions
func (o *InstallOptions) printSuccessMessage() {
	fmt.Println()
	fmt.Println("✓ Completion files generated successfully!")
	fmt.Println()

	o.printSetupInstructions()
}

// printSetupInstructions prints shell-specific setup instructions
func (o *InstallOptions) printSetupInstructions() {
	// Check if user already has completion setup in RC file
	hasCompletionSetup := o.hasExistingCompletionSetup()

	switch o.Shell {
	case shellFish:
		fmt.Println("✓ Setup complete! Fish auto-loads completions.")
		fmt.Println()
		fmt.Println("Restart fish (or run: exec fish) to activate completions.")

	case shellZsh:
		if hasCompletionSetup {
			fmt.Println("✓ Setup complete! Your ~/.zshrc has completion infrastructure.")
			fmt.Println()
			fmt.Println("Completions will work after restarting your shell:")
			fmt.Println("  source ~/.zshrc")
		} else {
			fmt.Println("⚠ Additional setup required!")
			fmt.Println()
			fmt.Println("Your ~/.zshrc is missing completion setup. Run:")
			fmt.Println()
			fmt.Println("cat >> ~/.zshrc << 'EOF'")
			fmt.Println("fpath=(~/.zsh/completions $fpath)")
			fmt.Println("autoload -Uz compinit && compinit")
			fmt.Println("EOF")
			fmt.Println()
			fmt.Println("Then restart your shell:\n source ~/.zshrc")
		}

	case shellBash:
		if hasCompletionSetup {
			fmt.Println("✓ Setup complete! Your ~/.bashrc has bash-completion loaded.")
			fmt.Println()
			fmt.Println("Completions will work after restarting your shell:")
			fmt.Println("  source ~/.bashrc")
		} else {
			fmt.Println("⚠ Additional setup required!")
			fmt.Println()
			fmt.Println("Your ~/.bashrc is missing completion setup. Run:")
			fmt.Println()
			fmt.Println("cat >> ~/.bashrc << 'EOF'")
			fmt.Println("# Load bash-completion framework")
			fmt.Println("if [ -f $(brew --prefix)/etc/bash_completion ]; then")
			fmt.Println("    . $(brew --prefix)/etc/bash_completion")
			fmt.Println("elif [ -f /etc/bash_completion ]; then")
			fmt.Println("    . /etc/bash_completion")
			fmt.Println("fi")
			fmt.Println()
			fmt.Println("# Load custom completions")
			fmt.Println("if [ -d ~/.local/share/bash-completion/completions ]; then")
			fmt.Println("    for f in ~/.local/share/bash-completion/completions/*; do")
			fmt.Println("        [ -r \"$f\" ] && . \"$f\"")
			fmt.Println("    done")
			fmt.Println("fi")
			fmt.Println("EOF")
			fmt.Println()
			fmt.Println("Then restart your shell:\n source ~/.bashrc")
		}
	}
}

// hasExistingCompletionSetup checks if RC file already has completion infrastructure
func (o *InstallOptions) hasExistingCompletionSetup() bool {
	config := shellConfigs[o.Shell]

	// Fish auto-loads, no RC file check needed
	if config.rcFile == "" {
		return config.completionCheck("")
	}

	rcFile := o.getRCFile()
	content, err := os.ReadFile(rcFile)
	if err != nil {
		return false
	}

	return config.completionCheck(string(content))
}

// NewInstallCommand creates the install subcommand
func NewInstallCommand() *cobra.Command {
	o := &InstallOptions{}

	c := &cobra.Command{
		Use:   "install",
		Short: "Install shell completions for oc and kubectl-oadp",
		Long: `Install shell completions for oc and kubectl-oadp.

Automatically detects your shell and sets up completions in the appropriate
directory and configuration file.

Supported shells: bash, zsh, fish

For bash, the bash-completion package must be installed first.`,
		Example: `  # Install completions for current shell
  kubectl-oadp completion install

  # Install completions for specific shell
  kubectl-oadp completion install --shell zsh

  # Reinstall/overwrite existing completions
  kubectl-oadp completion install --force`,
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(args); err != nil {
				return err
			}
			if err := o.Validate(); err != nil {
				return err
			}
			return o.Run(c)
		},
	}

	o.BindFlags(c.Flags())
	return c
}
