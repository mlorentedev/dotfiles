package sshtarget

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"slices"
	"strings"
)

// WindowsMeshInterface is the adapter the mesh arrives on under Windows: the
// Tailscale client names it Tailscale, where Linux names it tailscale0.
const WindowsMeshInterface = "Tailscale"

// sshdRule is the rule the OpenSSH.Server capability creates. It creates it
// open to every address, so the firewall resource narrows an existing rule
// rather than adding a closed one beside it.
const sshdRule = "OpenSSH-Server-In-TCP"

// WindowsFirewall closes port 22 to everything but the mesh on Windows: the
// sshd rule matches the Tailscale adapter AND the mesh range, because
// 100.64.0.0/10 is also carrier-grade NAT space and the address alone is not
// the mesh. Every read is a PowerShell query that needs no administrator
// rights, parsed as JSON rather than matched as text.
type WindowsFirewall struct {
	Run Runner // the user's reads
}

// fwState is what one read of the firewall returns.
type fwState struct {
	Adapter  bool     `json:"adapter"` // the mesh adapter exists
	Exists   bool     `json:"exists"`
	Enabled  string   `json:"enabled"`
	Action   string   `json:"action"`
	Remote   []string `json:"remote"`
	Iface    []string `json:"iface"`
	Disabled []string `json:"disabled"` // firewall profiles that are off
	Others   []fwRule `json:"others"`   // other enabled inbound Allow rules on TCP 22
}

type fwRule struct {
	Name   string   `json:"name"`
	Remote []string `json:"remote"`
	Iface  []string `json:"iface"`
}

// fwQuery reads the sshd rule, the profiles, and every other enabled inbound
// Allow rule that opens TCP 22. `@(...)` keeps a single value an array in the
// JSON, and ConvertTo-Json drops nothing at -Depth 4.
const fwQuery = `$r = Get-NetFirewallRule -Name '` + sshdRule + `' -ErrorAction SilentlyContinue
$o = [ordered]@{ adapter = $null -ne (Get-NetAdapter -Name '` + WindowsMeshInterface + `' -ErrorAction SilentlyContinue); exists = $null -ne $r; enabled = ''; action = ''; remote = @(); iface = @() }
if ($r) {
  $o.enabled = [string]$r.Enabled; $o.action = [string]$r.Action
  $o.remote = @(($r | Get-NetFirewallAddressFilter).RemoteAddress)
  $o.iface = @(($r | Get-NetFirewallInterfaceFilter).InterfaceAlias)
}
$o.disabled = @(Get-NetFirewallProfile | Where-Object { [string]$_.Enabled -ne 'True' } | ForEach-Object { [string]$_.Name })
$o.others = @(Get-NetFirewallPortFilter -Protocol TCP | Where-Object { @($_.LocalPort) -contains '22' } | Get-NetFirewallRule |
  Where-Object { [string]$_.Enabled -eq 'True' -and [string]$_.Direction -eq 'Inbound' -and [string]$_.Action -eq 'Allow' -and $_.Name -ne '` + sshdRule + `' } |
  ForEach-Object { [ordered]@{ name = $_.Name; remote = @(($_ | Get-NetFirewallAddressFilter).RemoteAddress); iface = @(($_ | Get-NetFirewallInterfaceFilter).InterfaceAlias) } })
$o | ConvertTo-Json -Compress -Depth 4`

func (w WindowsFirewall) state() (fwState, error) {
	var s fwState
	out, err := w.Run(psArgv(fwQuery)...)
	if err != nil {
		return s, fmt.Errorf("read the Windows firewall: %w: %s", err, strings.TrimSpace(string(out)))
	}
	if err := json.Unmarshal(out, &s); err != nil {
		return s, fmt.Errorf("read the Windows firewall: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return s, nil
}

// meshOnly reports whether a rule admits the mesh and nothing else: the
// mesh adapter, and remote addresses all inside the mesh range.
func meshOnly(remote, iface []string) bool {
	if len(iface) != 1 || !strings.EqualFold(iface[0], WindowsMeshInterface) || len(remote) == 0 {
		return false
	}
	_, mesh, _ := net.ParseCIDR(MeshCIDR)
	meshOnes, _ := mesh.Mask.Size()
	for _, r := range remote {
		n := firewallNet(r)
		if n == nil {
			return false
		}
		ones, bits := n.Mask.Size()
		if bits == 0 || !mesh.Contains(n.IP) || ones < meshOnes {
			return false
		}
	}
	return true
}

// firewallNet parses a RemoteAddress entry: an address, a CIDR, or the
// address/dotted-mask form Windows stores a subnet in (100.64.0.0/10 reads
// back as 100.64.0.0/255.192.0.0). Keywords such as Any or LocalSubnet, and
// ranges, return nil, and nil never counts as inside the mesh.
func firewallNet(r string) *net.IPNet {
	if _, n, err := net.ParseCIDR(r); err == nil {
		return n
	}
	addr, mask, found := strings.Cut(r, "/")
	ip := net.ParseIP(addr).To4()
	if ip == nil {
		return nil
	}
	if !found {
		return &net.IPNet{IP: ip, Mask: net.CIDRMask(32, 32)}
	}
	m := net.ParseIP(mask).To4()
	if m == nil {
		return nil
	}
	n := &net.IPNet{IP: ip.Mask(net.IPMask(m)), Mask: net.IPMask(m)}
	if _, bits := n.Mask.Size(); bits == 0 { // not a contiguous mask
		return nil
	}
	return n
}

// refusal names what the target will not change on the owner's behalf: no
// mesh adapter (a rule bound to an adapter that does not exist admits
// nothing, and the mesh address may be on another one), a profile that is off (a narrow rule on a disabled firewall fences nothing,
// and turning a whole profile on is the owner's call), and another rule that
// opens port 22 off the mesh.
func (s fwState) refusal() error {
	var why []string
	if !s.Adapter {
		why = append(why, "there is no "+WindowsMeshInterface+" network adapter to bind the rule to: join the mesh with the Tailscale client")
	}
	if len(s.Disabled) > 0 {
		why = append(why, "the firewall is off for the "+strings.Join(s.Disabled, ", ")+" profile(s): turn it on (Set-NetFirewallProfile -Enabled True)")
	}
	var open []string
	for _, r := range s.Others {
		if !meshOnly(r.Remote, r.Iface) {
			open = append(open, r.Name)
		}
	}
	if len(open) > 0 {
		slices.Sort(open)
		why = append(why, "rules "+strings.Join(open, ", ")+" open port 22 off the mesh: disable or narrow them")
	}
	if len(why) == 0 {
		return nil
	}
	return errors.New("the Windows firewall would still admit ssh from outside the mesh, and nothing was changed: " + strings.Join(why, "; "))
}

func (s fwState) converged() bool {
	return s.Exists && s.Enabled == "True" && s.Action == "Allow" && meshOnly(s.Remote, s.Iface)
}

func (w WindowsFirewall) Differs() (bool, error) {
	s, err := w.state()
	if err != nil {
		return false, err
	}
	return !s.converged() || s.refusal() != nil, nil
}

// Apply narrows the sshd rule to the mesh, or creates it narrow when the
// capability's rule is missing. It refuses before changing anything when
// another rule or a disabled profile would keep port 22 open anyway.
func (w WindowsFirewall) Apply(root Root) (func() error, error) {
	s, err := w.state()
	if err != nil {
		return nil, err
	}
	if err := s.refusal(); err != nil {
		return nil, err
	}
	if s.converged() {
		return nil, nil
	}
	narrow := fmt.Sprintf("-InterfaceAlias %s -RemoteAddress %s", psQuote(WindowsMeshInterface), psQuote(MeshCIDR))
	if !s.Exists {
		err := powershell(root, fmt.Sprintf("New-NetFirewallRule -Name %s -DisplayName 'OpenSSH Server (sshd)' -Enabled True -Direction Inbound -Protocol TCP -Action Allow -LocalPort 22 %s | Out-Null",
			psQuote(sshdRule), narrow))
		undo := func() error { return powershell(root, "Remove-NetFirewallRule -Name "+psQuote(sshdRule)) }
		if err != nil {
			return nil, err
		}
		return undo, nil
	}
	undo := func() error {
		return powershell(root, fmt.Sprintf("Set-NetFirewallRule -Name %s -Enabled %s -Action %s -InterfaceAlias %s -RemoteAddress %s",
			psQuote(sshdRule), orDefault(s.Enabled, "True"), orDefault(s.Action, "Allow"), psList(s.Iface, "Any"), psList(s.Remote, "Any")))
	}
	if err := powershell(root, fmt.Sprintf("Set-NetFirewallRule -Name %s -Enabled True -Action Allow %s", psQuote(sshdRule), narrow)); err != nil {
		return undo, err
	}
	return undo, nil
}

// Prove reads the firewall back: the rule admits the mesh only, every profile
// is on, and no other rule opens port 22.
func (w WindowsFirewall) Prove() error {
	s, err := w.state()
	if err != nil {
		return err
	}
	if err := s.refusal(); err != nil {
		return err
	}
	if !s.converged() {
		return fmt.Errorf("the %s rule does not admit the mesh only: enabled %q, action %q, interface %v, remote %v", sshdRule, s.Enabled, s.Action, s.Iface, s.Remote)
	}
	return nil
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// psList renders values as a PowerShell array argument, def when empty.
func psList(values []string, def string) string {
	if len(values) == 0 {
		return psQuote(def)
	}
	q := make([]string, len(values))
	for i, v := range values {
		q[i] = psQuote(v)
	}
	return "@(" + strings.Join(q, ",") + ")"
}
