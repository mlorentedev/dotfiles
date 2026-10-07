package harness

import "testing"

const enforcedV1 = "<!-- BEGIN HARNESS GENERATED (sha256:aaa) -->\nrule one\n<!-- END HARNESS GENERATED -->\n"
const enforcedV2 = "<!-- BEGIN HARNESS GENERATED (sha256:bbb) -->\nrule one\nrule two\n<!-- END HARNESS GENERATED -->\n"
const presence = "\n<!-- BEGIN HARNESS AGENT-PRESENCE (sha256:p1) -->\nroster\n<!-- END HARNESS AGENT-PRESENCE -->\n"
const catalog = "\n<!-- BEGIN HARNESS GENERATED (sha256:c1) — skill catalog -->\nskills\n<!-- END HARNESS GENERATED -->\n"

func TestDeployedMatchesSource(t *testing.T) {
	source := "# CLAUDE\n\n" + enforcedV1 + "\ntail\n"
	cases := []struct {
		name     string
		deployed string
		source   string
		want     bool
	}{
		{"a verbatim copy", source, source, true},
		{"deploy-only regions appended", source + presence + catalog, source, true},
		{"CRLF deployed, trailing newlines differ", "# CLAUDE\r\n\r\n" + "<!-- BEGIN HARNESS GENERATED (sha256:aaa) -->\r\nrule one\r\n<!-- END HARNESS GENERATED -->\r\n\r\ntail\r\n\r\n", source, true},
		// F-064: the source's own region was refreshed; the deployed copy
		// still carries the old one. Stripping every region on both sides
		// called this a match.
		{"a refreshed source region is drift", source + presence, "# CLAUDE\n\n" + enforcedV2 + "\ntail\n", false},
		{"text outside the regions differs", source + presence, "# CLAUDE v2\n\n" + enforcedV1 + "\ntail\n", false},
		{"the deployed copy lost a source region", "# CLAUDE\n\ntail\n" + presence, source, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := DeployedMatchesSource(tc.deployed, tc.source); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
