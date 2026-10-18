package vm

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func testMachine(dir string) Machine {
	return Machine{
		Layout:   Layout{Dir: dir},
		Firmware: "/opt/homebrew/share/qemu/edk2-aarch64-code.fd",
		CPUs:     4,
		MemoryMB: 8192,
	}
}

// The Windows guest must see nothing of the host: no shared directory (the
// host's ~ carries credentials, the working tree carries LF files a Windows
// checkout would rewrite), no display, and every host path it is given lies
// under its own state dir or is the read-only firmware.
func TestQEMUArgs_ShareNothingWithTheHost(t *testing.T) {
	dir := "/state/vm/windows"
	for _, m := range []Machine{testMachine(dir), withInstall(testMachine(dir))} {
		args, err := m.Args()
		if err != nil {
			t.Fatal(err)
		}
		joined := strings.Join(args, " ")
		for _, bad := range []string{"-virtfs", "-fsdev", "virtio-9p", "vhost-user-fs", "fat:rw", "-display gtk", "-display cocoa", "-vnc", "-spice", "hostfwd"} {
			if strings.Contains(joined, bad) {
				t.Errorf("argv carries %q: %s", bad, joined)
			}
		}
		if !hasPair(args, "-display", "none") {
			t.Error("the guest must run with -display none")
		}
		for _, p := range hostPaths(args) {
			if p != m.Firmware && !strings.HasPrefix(p, dir+"/") {
				t.Errorf("host path %s is outside the guest's state dir", p)
			}
		}
	}
}

// The agent channel is the host's only way in, and the guest needs the
// devices Windows 11 on Arm boots with: UEFI from read-only firmware, an NVMe
// disk Windows has a driver for, USB input, and HVF acceleration.
func TestQEMUArgs_BootWindowsOnArm(t *testing.T) {
	args, err := testMachine("/s").Args()
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	for _, want := range []string{
		"-accel hvf", "-cpu host", "virt,gic-version=3",
		"if=pflash,format=raw,unit=0,file=/opt/homebrew/share/qemu/edk2-aarch64-code.fd,readonly=on",
		"if=pflash,format=raw,unit=1,file=/s/run-vars.fd",
		"nvme,drive=disk0", "file=/s/run.qcow2,format=qcow2",
		"virtserialport,chardev=qga0,name=org.qemu.guest_agent.0",
		"socket,id=qga0,path=/s/run/qga.sock,server=on,wait=off",
		"-qmp unix:/s/run/qmp.sock,server=on,wait=off",
		"-smbios type=1,uuid=" + guestUUID,
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("argv lacks %q:\n%s", want, joined)
		}
	}
	// HVF refuses tpm-tis-device (HV_BAD_ARGUMENT, measured 2026-10-17):
	// a TPM here makes QEMU exit before the guest boots.
	if strings.Contains(joined, "-tpmdev") {
		t.Error("the guest must have no TPM device under HVF")
	}
}

// A run boots a per-run overlay over clean-base and a copy of its UEFI
// variables; only the build boots clean-base itself. clean-base is then never
// opened writable by a run, so nothing a run does can persist in it.
func TestQEMUArgs_ARunNeverWritesCleanBase(t *testing.T) {
	run, err := testMachine("/s").Args()
	if err != nil {
		t.Fatal(err)
	}
	if j := strings.Join(run, " "); strings.Contains(j, "clean-base") {
		t.Errorf("a run's argv names clean-base: %s", j)
	}
	for _, m := range []Machine{withBase(testMachine("/s")), withInstall(testMachine("/s"))} {
		args, err := m.Args()
		if err != nil {
			t.Fatal(err)
		}
		j := strings.Join(args, " ")
		for _, want := range []string{"file=/s/clean-base.qcow2,format=qcow2", "unit=1,file=/s/clean-base-vars.fd"} {
			if !strings.Contains(j, want) {
				t.Errorf("the build's argv lacks %q", want)
			}
		}
	}
}

// The install boots from the Windows ISO; every reboot after it falls through
// to the disk, because nothing presses a key at the CD's prompt then.
func TestQEMUArgs_InstallBootsTheISOFirst(t *testing.T) {
	args, err := withInstall(testMachine("/s")).Args()
	if err != nil {
		t.Fatal(err)
	}
	j := strings.Join(args, " ")
	for _, want := range []string{"usb-storage,drive=cd0,removable=on,bootindex=0", "nvme,drive=disk0,serial=dotf0,bootindex=1"} {
		if !strings.Contains(j, want) {
			t.Errorf("argv lacks %q", want)
		}
	}
}

// Install media are read-only and removable: Windows Setup looks for
// autounattend.xml on removable media, and nothing it writes reaches the host.
func TestQEMUArgs_InstallMediaAreReadOnly(t *testing.T) {
	args, err := withInstall(testMachine("/s")).Args()
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	for _, want := range []string{
		"file=/s/media/windows.iso,media=cdrom,readonly=on",
		"file=/s/media/virtio-win.iso,media=cdrom,readonly=on",
		"file=fat:ro:/s/media/unattend,readonly=on",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("argv lacks %q", want)
		}
	}
}

// macOS caps a unix socket path at 104 bytes; QEMU fails late and obscurely.
func TestQEMUArgs_RefusesASocketPathTooLong(t *testing.T) {
	if _, err := testMachine("/" + strings.Repeat("d", 100)).Args(); err == nil || !strings.Contains(err.Error(), "socket") {
		t.Fatalf("want an error naming the socket path, got %v", err)
	}
}

func withBase(m Machine) Machine {
	m.Base = true
	return m
}

func withInstall(m Machine) Machine {
	m.Install = true
	return m
}

func hasPair(args []string, k, v string) bool {
	i := slices.Index(args, k)
	return i >= 0 && i+1 < len(args) && args[i+1] == v
}

// hostPaths pulls every absolute path out of argv's key=value lists and
// socket specs.
func hostPaths(args []string) []string {
	var out []string
	for _, a := range args {
		for field := range strings.SplitSeq(a, ",") {
			_, v, ok := strings.Cut(field, "=")
			if !ok {
				v = field
			}
			v = strings.TrimPrefix(v, "unix:")
			v = strings.TrimPrefix(v, "fat:ro:")
			if filepath.IsAbs(v) {
				out = append(out, v)
			}
		}
	}
	return out
}
