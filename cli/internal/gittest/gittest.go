// Package gittest keeps a test process's git from forking a background gc that
// outlives the test (#2803). It works through the environment, not repo config,
// because that reaches clones and remotes too: see lesson 517.
package gittest

import "os"

const countVar = "GIT_CONFIG_COUNT"

var quiet = map[string]string{
	countVar:             "2",
	"GIT_CONFIG_KEY_0":   "gc.auto",
	"GIT_CONFIG_VALUE_0": "0",
	"GIT_CONFIG_KEY_1":   "maintenance.auto",
	"GIT_CONFIG_VALUE_1": "false",
}

// Quiet turns off auto gc and auto maintenance for every git this process
// starts. Call it from TestMain before m.Run. A test binary that re-executes
// itself inherits the keys, so finding exactly these is fine; any other
// GIT_CONFIG_COUNT panics, since overwriting it would drop the caller's keys.
func Quiet() {
	if v, ok := os.LookupEnv(countVar); ok {
		if alreadyQuiet() {
			return
		}
		panic("gittest: " + countVar + " is already set (" + v + ")")
	}
	for k, v := range quiet {
		if err := os.Setenv(k, v); err != nil {
			panic("gittest: " + err.Error())
		}
	}
}

func alreadyQuiet() bool {
	for k, v := range quiet {
		if os.Getenv(k) != v {
			return false
		}
	}
	return true
}
