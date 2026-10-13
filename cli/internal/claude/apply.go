package claude

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// ApplyMCP registers the servers in mcp-servers.json under repoRoot (CLI-063,
// #1339). It is the one implementation behind a bare `dotf deploy` and the
// converge step (#2387): a server that did not register is in the report and
// in the returned error, by name.
func ApplyMCP(s Syncer, repoRoot string, dryRun bool) (MCPReport, error) {
	servers, err := LoadServers(filepath.Join(repoRoot, ServersRel))
	if err != nil {
		return MCPReport{}, err
	}
	rep := s.Register(servers, dryRun)
	if len(rep.Failed) > 0 {
		err := fmt.Errorf("%d server(s) failed to register: %s", len(rep.Failed), strings.Join(rep.Failed, ", "))
		if len(rep.Lost) > 0 {
			err = fmt.Errorf("%w; removed and not restored: %s", err, strings.Join(rep.Lost, ", "))
		}
		return rep, err
	}
	return rep, nil
}

// PluginsResult is one ApplyPlugins: the retired marketplaces it removed, the
// declared plugins it installed, and how many were declared.
type PluginsResult struct {
	Retired  RetireReport
	Plugins  PluginReport
	Declared int
}

// ApplyPlugins removes the retired marketplaces in ai/claude/plugins.json, then
// installs the declared plugins a box lacks. The removal failing does not stop
// the installs; both failures are returned. A failed install is returned with
// what the CLI said for it, so a caller that prints only the error still names
// the cause (BUG-121, #2336).
func ApplyPlugins(s Syncer, repoRoot string, dryRun bool) (PluginsResult, error) {
	list, err := LoadPlugins(filepath.Join(repoRoot, filepath.FromSlash(PluginsRel)))
	if err != nil {
		return PluginsResult{}, err
	}
	s.Sources, err = LoadMarketplaceSources(filepath.Join(repoRoot, filepath.FromSlash(MarketplacesRel)))
	if err != nil {
		return PluginsResult{}, err
	}
	res := PluginsResult{Declared: len(list.Plugins)}
	var retireErr error
	res.Retired, retireErr = s.Retire(list.RetiredMarketplaces, dryRun)
	if retireErr == nil && len(res.Retired.Failed) > 0 {
		retireErr = fmt.Errorf("retired marketplace(s) still registered: %s", strings.Join(res.Retired.Failed, ", "))
	}
	res.Plugins, err = s.Sync(list.Plugins, dryRun)
	if err != nil {
		return res, errors.Join(retireErr, err)
	}
	if err := accountedFor(res.Plugins, res.Declared); err != nil {
		return res, errors.Join(retireErr, err)
	}
	if n := len(res.Plugins.Failed); n > 0 {
		causes := make([]string, 0, n)
		for _, id := range res.Plugins.Failed {
			causes = append(causes, fmt.Sprintf("%s (%v)", id, res.Plugins.Causes[id]))
		}
		return res, errors.Join(retireErr, fmt.Errorf("%d plugin(s) failed to install: %s", n, strings.Join(causes, "; ")))
	}
	return res, retireErr
}

// accountedFor fails a report that does not place every declared plugin in
// exactly one of Present, Added and Failed: a summary that counts fewer than
// were declared would read as success while some were never tried (#2336).
func accountedFor(rep PluginReport, declared int) error {
	if n := len(rep.Present) + len(rep.Added) + len(rep.Failed); n != declared {
		return fmt.Errorf("the report accounts for %d of %d declared plugins", n, declared)
	}
	return nil
}
