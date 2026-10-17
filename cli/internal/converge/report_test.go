package converge

import (
	"os"
	"path/filepath"
	"testing"
)

// What WriteReport persists, ParseRunReport reads back: the guest's report is
// read by a host process, so the two ends must agree on the format.
func TestParseRunReport_ReadsWhatWriteReportWrote(t *testing.T) {
	path := filepath.Join(t.TempDir(), "last.json")
	rep := Report{GOOS: "linux", Entries: []Entry{{Name: "tools", Status: StatusChange, Changes: 2}, {Name: "configs", Status: StatusOK}}}
	if err := WriteReport(path, rep, nil); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseRunReport(data)
	if err != nil {
		t.Fatal(err)
	}
	if got.Result != "ok" || got.Changed != 2 || len(got.Entries) != 2 || got.Entries[0].Name != "tools" {
		t.Errorf("round trip lost data: %+v", got)
	}
}

func TestParseRunReport_RejectsWhatIsNotAReport(t *testing.T) {
	for _, in := range []string{"", "{}", `{"result":"maybe"}`, "not json"} {
		if _, err := ParseRunReport([]byte(in)); err == nil {
			t.Errorf("%q parsed as a converge report", in)
		}
	}
}
