package vm

import (
	"fmt"
	"path/filepath"
	"strconv"
)

// guestUUID is the guest's SMBIOS system UUID. It is fixed so the guest's
// hardware identity, which Windows activation binds a digital licence to,
// survives every revert.
const guestUUID = "5d0f7a3c-2b8e-4c61-9a4e-7f1d3c2b8e10"

// maxSocketPath is the longest unix socket path macOS accepts (sun_path is
// 104 bytes, NUL included).
const maxSocketPath = 103

// Layout is where one guest's files live on the host: everything under one
// state dir, so nothing the guest is given lies elsewhere.
type Layout struct{ Dir string }

// BaseDisk and BaseVars are clean-base: the installed guest the build leaves,
// read-only once it is complete. Disk and Vars are one run's: an overlay over
// BaseDisk and a copy of BaseVars, thrown away after the run.
func (l Layout) BaseDisk() string { return filepath.Join(l.Dir, "clean-base.qcow2") }
func (l Layout) BaseVars() string { return filepath.Join(l.Dir, "clean-base-vars.fd") }
func (l Layout) Disk() string     { return filepath.Join(l.Dir, "run.qcow2") }
func (l Layout) Vars() string     { return filepath.Join(l.Dir, "run-vars.fd") }
func (l Layout) Log() string      { return filepath.Join(l.Dir, "qemu.log") }
func (l Layout) Stuck() string    { return filepath.Join(l.Dir, "build-stuck.png") }
func (l Layout) Media() string    { return filepath.Join(l.Dir, "media") }
func (l Layout) WindowsISO() string {
	return filepath.Join(l.Media(), "windows.iso")
}
func (l Layout) VirtioISO() string   { return filepath.Join(l.Media(), "virtio-win.iso") }
func (l Layout) UnattendDir() string { return filepath.Join(l.Media(), "unattend") }
func (l Layout) QGASocket() string   { return filepath.Join(l.Dir, "run", "qga.sock") }
func (l Layout) QMPSocket() string   { return filepath.Join(l.Dir, "run", "qmp.sock") }

// Machine is the Windows 11 on Arm guest QEMU runs. dotf owns its definition
// (ADR-053): what Windows needs to boot, and nothing that reaches the host.
// It has no TPM: under HVF, upstream QEMU cannot map the virt machine's only
// TPM device (tpm-tis-device fails with HV_BAD_ARGUMENT), so Setup is told
// to skip the check instead.
type Machine struct {
	Layout   Layout
	Firmware string // edk2-aarch64-code.fd, read-only
	CPUs     int
	MemoryMB int
	Base     bool // boot clean-base itself, which only the build does; otherwise a run's overlay
	Install  bool // attach the Windows and virtio-win ISOs and the unattend media; implies Base
}

// Args is QEMU's command line for the machine.
func (m Machine) Args() ([]string, error) {
	l := m.Layout
	disk, vars := l.Disk(), l.Vars()
	if m.Base || m.Install {
		disk, vars = l.BaseDisk(), l.BaseVars()
	}
	for _, s := range []string{l.QGASocket(), l.QMPSocket()} {
		if len(s) > maxSocketPath {
			return nil, fmt.Errorf("socket path %s is %d bytes, over macOS's %d: move the guest's state dir somewhere shorter", s, len(s), maxSocketPath)
		}
	}
	args := []string{
		"-nodefaults", "-no-user-config",
		"-machine", "virt,gic-version=3",
		"-accel", "hvf", "-cpu", "host",
		"-smp", strconv.Itoa(m.CPUs), "-m", strconv.Itoa(m.MemoryMB),
		"-smbios", "type=1,uuid=" + guestUUID,
		"-drive", "if=pflash,format=raw,unit=0,file=" + m.Firmware + ",readonly=on",
		"-drive", "if=pflash,format=raw,unit=1,file=" + vars,
		"-device", "ramfb",
		"-device", "qemu-xhci,id=usb",
		"-device", "usb-kbd", "-device", "usb-tablet",
		"-drive", "if=none,id=disk0,file=" + disk + ",format=qcow2,discard=unmap",
		"-device", "nvme,drive=disk0,serial=dotf0,bootindex=1",
		"-netdev", "user,id=net0",
		"-device", "virtio-net-pci,netdev=net0",
		"-device", "virtio-serial-pci",
		"-chardev", "socket,id=qga0,path=" + l.QGASocket() + ",server=on,wait=off",
		"-device", "virtserialport,chardev=qga0,name=org.qemu.guest_agent.0",
		"-qmp", "unix:" + l.QMPSocket() + ",server=on,wait=off",
		"-display", "none",
	}
	if m.Install {
		args = append(args, installMedia(l)...)
	}
	return args, nil
}

// installMedia attaches the two ISOs as USB CD drives and the unattend
// directory as a read-only USB stick: Windows Setup reads autounattend.xml
// from removable media. The Windows ISO boots before the disk; its "press any
// key" prompt is answered only at the first boot, so Setup's reboots fall
// through to the disk.
func installMedia(l Layout) []string {
	var args []string
	for i, iso := range []string{l.WindowsISO(), l.VirtioISO()} {
		id := "cd" + strconv.Itoa(i)
		boot := ""
		if i == 0 {
			boot = ",bootindex=0"
		}
		args = append(args,
			"-drive", "if=none,id="+id+",file="+iso+",media=cdrom,readonly=on",
			"-device", "usb-storage,drive="+id+",removable=on"+boot)
	}
	return append(args,
		"-drive", "if=none,id=unattend,format=raw,file=fat:ro:"+l.UnattendDir()+",readonly=on",
		"-device", "usb-storage,drive=unattend,removable=on")
}
