//go:build !windows

package env

import "testing"

func TestRegisteredPath_IsEmptyOffWindows(t *testing.T) {
	if dirs, err := RegisteredPath(); dirs != nil || err != nil {
		t.Errorf("RegisteredPath() = %v, %v; want nothing", dirs, err)
	}
}
