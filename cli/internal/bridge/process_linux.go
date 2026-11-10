//go:build linux

package bridge

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func observeProcess(pid int) (processSnapshot, error) {
	if pid <= 0 {
		return processSnapshot{}, errProcessNotFound
	}
	root := filepath.Join("/proc", fmt.Sprintf("%d", pid))
	executable, err := os.Readlink(filepath.Join(root, "exe"))
	if err != nil {
		if os.IsNotExist(err) {
			return processSnapshot{}, errProcessNotFound
		}
		return processSnapshot{}, err
	}
	raw, err := os.ReadFile(filepath.Join(root, "stat"))
	if err != nil {
		if os.IsNotExist(err) {
			return processSnapshot{}, errProcessNotFound
		}
		return processSnapshot{}, err
	}
	startIdentity, err := linuxStartIdentity(string(raw))
	if err != nil {
		return processSnapshot{}, err
	}
	return processSnapshot{
		Executable:    strings.TrimSuffix(executable, " (deleted)"),
		StartIdentity: startIdentity,
	}, nil
}

func linuxStartIdentity(stat string) (string, error) {
	closeParen := strings.LastIndex(stat, ")")
	if closeParen < 0 {
		return "", errors.New("process stat lacks command terminator")
	}
	fields := strings.Fields(stat[closeParen+1:])
	if len(fields) <= 19 {
		return "", fmt.Errorf("process stat has %d fields after command, want at least 20", len(fields))
	}
	return "linux:" + fields[19], nil
}
