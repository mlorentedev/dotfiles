package sshtarget

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// powercfgQ renders `powercfg /q` for one setting as Windows prints it, in
// English or Spanish: the labels change with the language, the hex values do
// not. A range setting (the idle timeouts) lists its bounds in hex first.
func powercfgQ(lang, setting string, ac, dc uint64) string {
	label := map[string][2]string{
		"en": {"Current AC Power Setting Index", "Current DC Power Setting Index"},
		"es": {"Índice de configuración de corriente alterna actual", "Índice de configuración de corriente continua actual"},
	}[lang]
	head := "Power Scheme GUID: 381b4222-f694-41f0-9685-ff5bb260df2e  (Balanced)\r\n" +
		"  Subgroup GUID: 238c9fa8-0aad-41ed-83f4-97be242c8f20  (Sleep)\r\n" +
		"    Power Setting GUID: 29f6c1db-86da-48c5-9fdb-f2b67b1f44da  (" + setting + ")\r\n" +
		"      GUID Alias: " + setting + "\r\n"
	if setting == "LIDACTION" {
		head += "      Possible Setting Index: 000\r\n      Possible Setting Friendly Name: Do nothing\r\n" +
			"      Possible Setting Index: 001\r\n      Possible Setting Friendly Name: Sleep\r\n"
	} else {
		head += "    Minimum Possible Setting: 0x00000000\r\n    Maximum Possible Setting: 0xffffffff\r\n" +
			"    Possible Settings increment: 0x00000001\r\n    Possible Settings units: Seconds\r\n"
	}
	return head + fmt.Sprintf("    %s: 0x%08x\r\n    %s: 0x%08x\r\n\r\n", label[0], ac, label[1], dc)
}

// fakeWin holds the active scheme's AC and DC values and applies powercfg run
// elevated. failAt, when set, fails the /setacvalueindex for that setting;
// failAfterWrite makes that failure come after the value was written.
type fakeWin struct {
	t              *testing.T
	lang           string
	ac, dc         map[string]uint64
	failAt         string
	failAfterWrite bool
	activated      int
	rootCalls      []string
}

func newFakeWin(t *testing.T, lang string) *fakeWin {
	// A laptop's Balanced scheme out of the box: sleep after 30 min, hibernate
	// after 3 h, sleep on lid close, on AC and battery alike.
	return &fakeWin{t: t, lang: lang,
		ac: map[string]uint64{"STANDBYIDLE": 1800, "HIBERNATEIDLE": 10800, "LIDACTION": 1},
		dc: map[string]uint64{"STANDBYIDLE": 900, "HIBERNATEIDLE": 10800, "LIDACTION": 1}}
}

func (w *fakeWin) user(argv ...string) ([]byte, error) {
	if len(argv) != 5 || argv[0] != "powercfg" || argv[1] != "/q" || argv[2] != "SCHEME_CURRENT" {
		w.t.Fatalf("unexpected user command %v", argv)
	}
	return []byte(powercfgQ(w.lang, argv[4], w.ac[argv[4]], w.dc[argv[4]])), nil
}

func (w *fakeWin) Ready() error { return nil }

func (w *fakeWin) Run(argv ...string) ([]byte, error) {
	w.rootCalls = append(w.rootCalls, strings.Join(argv, " "))
	switch {
	case slices.Equal(argv, []string{"powercfg", "/setactive", "SCHEME_CURRENT"}):
		w.activated++
	case len(argv) == 6 && argv[1] == "/setacvalueindex":
		if argv[4] == w.failAt && !w.failAfterWrite {
			return []byte("Invalid Parameters"), errors.New("exit status 1")
		}
		v, err := strconv.ParseUint(argv[5], 10, 64)
		if err != nil {
			w.t.Fatalf("non-decimal value in %v", argv)
		}
		w.ac[argv[4]] = v
		if argv[4] == w.failAt {
			return []byte("Access is denied."), errors.New("exit status 1")
		}
	default:
		w.t.Fatalf("unexpected elevated command %v", argv)
	}
	return nil, nil
}

func TestPowercfgSetsTheACProfileAndActivatesTheScheme(t *testing.T) {
	for _, lang := range []string{"en", "es"} {
		w := newFakeWin(t, lang)
		p := Powercfg{Run: w.user}
		if d, err := p.Differs(); err != nil || !d {
			t.Fatalf("%s: Differs() = %v, %v on a stock laptop, want true", lang, d, err)
		}
		if _, err := p.Apply(w); err != nil {
			t.Fatal(err)
		}
		if w.activated != 1 || !strings.HasSuffix(w.rootCalls[len(w.rootCalls)-1], "/setactive SCHEME_CURRENT") {
			t.Errorf("%s: the scheme must be activated once, after the values: %q", lang, w.rootCalls)
		}
		if err := p.Prove(); err != nil {
			t.Fatalf("%s: Prove after Apply: %v", lang, err)
		}
		if d, err := p.Differs(); err != nil || d {
			t.Errorf("%s: Differs() = %v, %v after Apply, want false: a second run must change nothing", lang, d, err)
		}
	}
}

// Battery keeps its values: a laptop that ignores its lid on battery drains.
func TestPowercfgLeavesBatteryAlone(t *testing.T) {
	w := newFakeWin(t, "en")
	if _, err := (Powercfg{Run: w.user}).Apply(w); err != nil {
		t.Fatal(err)
	}
	if w.dc["LIDACTION"] != 1 || w.dc["STANDBYIDLE"] != 900 {
		t.Errorf("battery values changed: %v", w.dc)
	}
	for _, c := range w.rootCalls {
		if strings.Contains(c, "setdcvalueindex") || strings.Contains(c, "/change") {
			t.Errorf("ran %q, which reaches battery", c)
		}
	}
}

func TestPowercfgConvergedRunsNothingElevated(t *testing.T) {
	w := newFakeWin(t, "en")
	for k := range w.ac {
		w.ac[k] = 0
	}
	if _, err := (Powercfg{Run: w.user}).Apply(w); err != nil {
		t.Fatal(err)
	}
	if len(w.rootCalls) != 0 {
		t.Errorf("a converged scheme ran %q", w.rootCalls)
	}
}

// A failure part way returns the undo of what it set; the undo puts back the
// values read before and activates the scheme again.
func TestPowercfgUndoRestoresTheValuesItRead(t *testing.T) {
	w := newFakeWin(t, "en")
	w.failAt = "LIDACTION"
	undo, err := (Powercfg{Run: w.user}).Apply(w)
	if err == nil {
		t.Fatal("Apply succeeded on a failing powercfg")
	}
	if w.ac["STANDBYIDLE"] != 0 {
		t.Fatal("the fake did not set the first value before failing")
	}
	if err := undo(); err != nil {
		t.Fatal(err)
	}
	if w.ac["STANDBYIDLE"] != 1800 || w.ac["HIBERNATEIDLE"] != 10800 || w.ac["LIDACTION"] != 1 {
		t.Errorf("after the undo: %v, want the stock values", w.ac)
	}
	if w.activated != 1 {
		t.Errorf("the undo activated the scheme %d times, want once", w.activated)
	}
}

// A write that fails after powercfg persisted it is undone too: the undo
// covers every setting Apply tried, not only those that reported success.
func TestPowercfgUndoRestoresAWriteThatFailedAfterPersisting(t *testing.T) {
	w := newFakeWin(t, "en")
	w.failAt, w.failAfterWrite = "LIDACTION", true
	undo, err := (Powercfg{Run: w.user}).Apply(w)
	if err == nil {
		t.Fatal("Apply succeeded on a failing powercfg")
	}
	if w.ac["LIDACTION"] != 0 {
		t.Fatal("the fake did not persist the failing write")
	}
	w.failAt = ""
	if err := undo(); err != nil {
		t.Fatal(err)
	}
	if w.ac["LIDACTION"] != 1 {
		t.Errorf("after the undo LIDACTION = %#x, want the stock 0x1", w.ac["LIDACTION"])
	}
}

func TestPowercfgProofNamesWhatStillDiffers(t *testing.T) {
	w := newFakeWin(t, "en")
	err := (Powercfg{Run: w.user}).Prove()
	if err == nil || !strings.Contains(err.Error(), "LIDACTION 0x1 (want 0x0)") {
		t.Fatalf("Prove = %v, want it to name LIDACTION", err)
	}
}

func TestParsePowercfgRefusesOutputWithoutIndexes(t *testing.T) {
	if _, err := parsePowercfgAC([]byte("Invalid Parameters -- try \"/?\" for help\r\n")); err == nil {
		t.Fatal("parsed an error message as a setting")
	}
	// A range setting's bounds are hex too: without the current indexes after
	// them, the maximum must not be read as the AC value.
	full := powercfgQ("en", "STANDBYIDLE", 1800, 900)
	bounds := full[:strings.Index(full, "    Current AC")]
	if v, err := parsePowercfgAC([]byte(bounds)); err == nil {
		t.Fatalf("parsed %#x from the range bounds alone", v)
	}
}
