package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"time"

	bridgepkg "github.com/mlorentedev/dotfiles/cli/internal/bridge"
	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/spf13/cobra"
)

type bridgeDeps struct {
	StateDir    func() (string, error)
	RoleEnabled func() (bool, error)
	Inspect     func(bridgepkg.ProcessRecord) (bridgepkg.ProcessInspection, error)
	Probe       func(bridgepkg.LaunchDescriptor) bridgepkg.Health
}

func newBridgeCmd() *cobra.Command {
	return newBridgeCmdWithDeps(bridgeDeps{
		StateDir: env.StateDir,
		RoleEnabled: func() (bool, error) {
			roles, err := env.LoadMachineRoles(env.MachinePath(env.Home()))
			return slices.Contains(roles, env.RoleTSBridgeClient), err
		},
		Inspect: bridgepkg.InspectProcess,
		Probe: func(launch bridgepkg.LaunchDescriptor) bridgepkg.Health {
			return bridgepkg.ProbeHealth(runBridgeProbe, launch)
		},
	})
}

func newBridgeCmdWithDeps(deps bridgeDeps) *cobra.Command {
	command := &cobra.Command{
		Use:           "bridge",
		Short:         "Operate the role-gated userspace bridge",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          func(c *cobra.Command, _ []string) error { return c.Help() },
	}
	command.AddCommand(newBridgeStatusCmd(deps))
	return command
}

func newBridgeStatusCmd(deps bridgeDeps) *cobra.Command {
	var profile string
	var jsonOut bool
	command := &cobra.Command{
		Use:           "status",
		Short:         "Report bridge process, listener and destination health",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(command *cobra.Command, _ []string) error {
			return runBridgeStatus(command, deps, profile, jsonOut)
		},
	}
	command.Flags().StringVar(&profile, "profile", "", "stored bridge profile (required when more than one exists)")
	command.Flags().BoolVar(&jsonOut, "json", false, "write the structured lifecycle verdict as JSON")
	return command
}

func runBridgeStatus(command *cobra.Command, deps bridgeDeps, profile string, jsonOut bool) error {
	if err := requireBridgeRole(command, deps); err != nil {
		return err
	}
	service, err := bridgeStatusService(command, deps)
	if err != nil {
		return err
	}
	result, err := service.Status(profile)
	if err != nil {
		return writeBridgeFailure(command, "bridge status is unavailable", err.Error(),
			"run `dotf bridge start --profile <name>` with the profile launch settings")
	}
	return writeBridgeStatus(command, result, jsonOut)
}

func requireBridgeRole(command *cobra.Command, deps bridgeDeps) error {
	if deps.RoleEnabled == nil {
		return writeBridgeFailure(command, "bridge role cannot be checked",
			"no machine-role reader is configured", "repair the dotf installation")
	}
	enabled, err := deps.RoleEnabled()
	if err != nil {
		return writeBridgeFailure(command, "bridge role cannot be checked", err.Error(),
			"repair machine.json, then retry")
	}
	if !enabled {
		return writeBridgeFailure(command, "bridge lifecycle is not enabled",
			"this machine does not declare the ts-bridge-client role",
			"run `dotf env role add ts-bridge-client`, then converge the role")
	}
	return nil
}

func bridgeStatusService(command *cobra.Command, deps bridgeDeps) (bridgepkg.Service, error) {
	if deps.StateDir == nil {
		return bridgepkg.Service{}, writeBridgeFailure(command, "bridge state cannot be located",
			"no state-directory resolver is configured", "repair the dotf installation")
	}
	stateDir, err := deps.StateDir()
	if err != nil {
		return bridgepkg.Service{}, writeBridgeFailure(command, "bridge state cannot be located", err.Error(),
			"set an absolute HOME or XDG_STATE_HOME, then retry")
	}
	return bridgepkg.Service{
		Store:   bridgepkg.NewStore(stateDir),
		Inspect: deps.Inspect,
		Probe:   deps.Probe,
	}, nil
}

func writeBridgeStatus(command *cobra.Command, result bridgepkg.StatusResult, jsonOut bool) error {
	if jsonOut {
		if err := json.NewEncoder(command.OutOrStdout()).Encode(result); err != nil {
			return err
		}
	} else if result.State == bridgepkg.StateHealthy {
		_, _ = fmt.Fprintf(command.OutOrStdout(), "bridge %s: healthy\n", result.Profile)
	} else {
		writeErrorWhyFix(command, result.Error, result.Why, result.Fix)
	}
	if result.State != bridgepkg.StateHealthy {
		return withExitCode(1, errors.New("bridge is not healthy"))
	}
	return nil
}

func writeBridgeFailure(command *cobra.Command, problem, why, fix string) error {
	writeErrorWhyFix(command, problem, why, fix)
	return withExitCode(1, errors.New(problem))
}

func writeErrorWhyFix(command *cobra.Command, problem, why, fix string) {
	_, _ = fmt.Fprintf(command.ErrOrStderr(), "ERROR: %s\nWHY: %s\nFIX: %s\n", problem, why, fix)
}

func runBridgeProbe(timeout time.Duration, name string, args ...string) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var stdout, stderr bytes.Buffer
	command := exec.CommandContext(ctx, name, args...)
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	if ctx.Err() != nil {
		err = ctx.Err()
	}
	return stdout.String(), stderr.String(), err
}
