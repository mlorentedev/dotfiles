package harness

import "testing"

func TestStripRegions(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "no markers -> unchanged",
			in:   "line1\nline2\n",
			want: "line1\nline2\n",
		},
		{
			name: "strips a GENERATED region",
			in: "before\n" +
				"<!-- BEGIN HARNESS GENERATED (sha256:abc) -->\n" +
				"injected content\n" +
				"<!-- END HARNESS GENERATED -->\n" +
				"after\n",
			want: "before\nafter\n",
		},
		{
			name: "strips an AGENT-PRESENCE region",
			in: "before\n" +
				"<!-- BEGIN HARNESS AGENT-PRESENCE (sha256:abc) -->\n" +
				"persona block\n" +
				"<!-- END HARNESS AGENT-PRESENCE -->\n" +
				"after\n",
			want: "before\nafter\n",
		},
		{
			name: "strips both region kinds, order-independent",
			in: "head\n" +
				"<!-- BEGIN HARNESS GENERATED (sha256:x) -->\nA\n<!-- END HARNESS GENERATED -->\n" +
				"mid\n" +
				"<!-- BEGIN HARNESS AGENT-PRESENCE (sha256:y) -->\nB\n<!-- END HARNESS AGENT-PRESENCE -->\n" +
				"tail\n",
			want: "head\nmid\ntail\n",
		},
		{
			// inject_agent_presence / replace_region's append branch write a
			// region as "\n" + BEGIN + body + END + "\n" onto untouched
			// content -- the blank line right before BEGIN must not survive
			// the strip, or a freshly-appended region reads as drift forever.
			name: "drops the blank separator line an appended region leaves behind",
			in:   "shared content\n" + "\n<!-- BEGIN HARNESS AGENT-PRESENCE (sha256:abc) -->\npersona\n<!-- END HARNESS AGENT-PRESENCE -->\n",
			want: "shared content\n",
		},
		{
			name: "a genuine blank line NOT before a region is preserved",
			in:   "para one\n\npara two\n",
			want: "para one\n\npara two\n",
		},
		{
			// A deployed copy written CRLF by a Windows tool (WIN-008/#1289):
			// the END marker must still close the region — a "\r"-suffixed
			// marker used to leave skip mode on to EOF — and no "\r" survives
			// into the comparison.
			name: "CRLF input: the region closes and line endings normalise",
			in:   "before\r\n<!-- BEGIN HARNESS GENERATED (sha256:abc) -->\r\ninjected\r\n<!-- END HARNESS GENERATED -->\r\nafter\r\n",
			want: "before\nafter\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := StripRegions(tc.in)
			if got != tc.want {
				t.Errorf("StripRegions(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
