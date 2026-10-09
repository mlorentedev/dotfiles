package spec

import (
	"fmt"
	"strings"
	"testing"
)

// signerPool is the shape amendment B (#1923) runs on: NaN first signers of two
// vendors, an Anthropic fallback and an Anthropic second signer.
const signerPool = `{"pool":[
 {"id":"nan/mimo-v2.6-flash","vendor":"xiaomi"},
 {"id":"nan/deepseek-v4-flash","vendor":"deepseek"},
 {"id":"anthropic-review/claude-haiku-5-5","vendor":"anthropic","signs":"fallback"},
 {"id":"anthropic-review/claude-sonnet-5-5","vendor":"anthropic","signs":"second"},
 {"id":"nan/glm5.3-flash","vendor":"zhipu","signs":"second"}
]}`

const highRisk = "---\nstatus: verifying\nrisk: high\n---\nclean\n"

// signedSpec lays down a spec with a first signature, an optional second one,
// and the launch requests the gate cross-checks them against.
func signedSpec(t *testing.T, root, proposal, first, firstReason, second string) {
	t.Helper()
	id := "AI-001-x"
	files := map[string]string{"proposal.md": proposal, ReviewFile: reviewBy(id, first)}
	files[ReviewRequestFile] = requestFor(first, firstReason)
	if second != "" {
		files[SecondSigner.Review] = reviewBy(id, second)
		files[SecondSigner.Request] = requestFor(second, "")
	}
	writeSpec(t, root, id, files)
}

func requestFor(reviewer, reason string) string {
	return fmt.Sprintf("{\"reviewed_sha\": %q, \"reviewer\": %q, \"fallback_reason\": %q}\n",
		strings.Repeat("0", 40), reviewer, reason)
}

func archiveErr(root string) error {
	_, err := Archive(root, "AI-001-x", ArchiveOptions{Staleness: fakeStaleness{}})
	return err
}

func TestSignerGate(t *testing.T) {
	cases := []struct {
		name                    string
		proposal, first, reason string
		second                  string
		wantErr                 string // "" means the archive succeeds
	}{
		{"a first signer alone archives a normal spec", "---\nstatus: verifying\n---\n", "nan/mimo-v2.6-flash", "", "", ""},
		{"a second-only member cannot sign first", "---\nstatus: verifying\n---\n", "anthropic-review/claude-sonnet-5-5", "", "", "only gives a second signature"},
		{"a fallback without a recorded reason is refused", "---\nstatus: verifying\n---\n", "anthropic-review/claude-haiku-5-5", "", "", "no classified reason"},
		{"a fallback with an unknown reason is refused", "---\nstatus: verifying\n---\n", "anthropic-review/claude-haiku-5-5", "flaky", "", "no classified reason"},
		{"a fallback with a classified reason archives", "---\nstatus: verifying\n---\n", "anthropic-review/claude-haiku-5-5", "rate-limit", "", ""},
		{"risk: high without a second signature is refused", highRisk, "nan/mimo-v2.6-flash", "", "", "risk: high"},
		{"risk: high with a cross-vendor second archives", highRisk, "nan/mimo-v2.6-flash", "", "anthropic-review/claude-sonnet-5-5", ""},
		{"a second signer of the first signer's vendor is refused", highRisk, "nan/mimo-v2.6-flash", "", "nan/mimo-v2.6-flash", "does not declare `signs: second`"},
		{"a first signer cannot give the second signature", highRisk, "nan/mimo-v2.6-flash", "", "nan/deepseek-v4-flash", "does not declare `signs: second`"},
		{"a voluntary second signature is checked too", "---\nstatus: verifying\n---\n", "nan/mimo-v2.6-flash", "", "nan/glm5.3-flash", ""},
		{"a second signer outside the pool is refused", highRisk, "nan/mimo-v2.6-flash", "", "claude-opus-5", "not in " + ReviewerPoolFile},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			writePool(t, root, signerPool)
			signedSpec(t, root, c.proposal, c.first, c.reason, c.second)
			err := archiveErr(root)
			switch {
			case c.wantErr == "" && err != nil:
				t.Fatalf("want the archive to succeed, got: %v", err)
			case c.wantErr != "" && err == nil:
				t.Fatalf("want a refusal naming %q, the archive succeeded", c.wantErr)
			case c.wantErr != "" && !strings.Contains(err.Error(), c.wantErr):
				t.Fatalf("want a refusal naming %q, got: %v", c.wantErr, err)
			}
		})
	}
}

// Two signatures of one vendor are one opinion. The pool marks both members
// `signs: second`-capable on the second side, so only the vendor check stops it.
func TestSignerGateRefusesASameVendorSecondSignature(t *testing.T) {
	root := t.TempDir()
	writePool(t, root, `{"pool":[
 {"id":"nan/mimo-v2.6-flash","vendor":"xiaomi"},
 {"id":"nan/mimo-pro","vendor":"xiaomi","signs":"second"}]}`)
	signedSpec(t, root, highRisk, "nan/mimo-v2.6-flash", "", "nan/mimo-pro")
	if err := archiveErr(root); err == nil || !strings.Contains(err.Error(), "another vendor") {
		t.Fatalf("want a same-vendor refusal, got: %v", err)
	}
}

// A FAIL from the second signer blocks exactly as a first-signer FAIL does.
func TestSignerGateASecondFailBlocks(t *testing.T) {
	root := t.TempDir()
	writePool(t, root, signerPool)
	signedSpec(t, root, highRisk, "nan/mimo-v2.6-flash", "", "anthropic-review/claude-sonnet-5-5")
	failing := strings.Replace(reviewBy("AI-001-x", "anthropic-review/claude-sonnet-5-5"), `"PASS"`, `"FAIL"`, 1)
	writeSpec(t, root, "AI-001-x", map[string]string{SecondSigner.Review: failing, SecondSigner.Request: requestFor("anthropic-review/claude-sonnet-5-5", "")})
	if err := archiveErr(root); err == nil || !strings.Contains(err.Error(), SecondSigner.Review+" records verdict FAIL") {
		t.Fatalf("want the second FAIL to block, got: %v", err)
	}
}

// Amendment B at load time: an Anthropic entry that would sign first makes the
// pool unreadable, whether it declares the vendor or only names the model.
func TestPoolRefusesAnAnthropicFirstSigner(t *testing.T) {
	for name, entry := range map[string]string{
		"declared vendor":  `{"id":"x/haiku","vendor":"anthropic"}`,
		"inferred from id": `{"id":"anthropic-review/claude-haiku-5-5"}`,
		"explicit first":   `{"id":"anthropic-review/claude-haiku-5-5","signs":"first"}`,
		"unknown signs":    `{"id":"nan/mimo-v2.6-flash","signs":"third"}`,
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writePool(t, root, `{"pool":[{"id":"nan/deepseek-v4-flash"},`+entry+`]}`)
			if _, err := LoadReviewerPoolEntries(root); err == nil {
				t.Fatal("want the pool refused")
			}
		})
	}
}

func signerEntries(t *testing.T) []ReviewerEntry {
	t.Helper()
	root := t.TempDir()
	writePool(t, root, signerPool)
	e, err := LoadReviewerPoolEntries(root)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

// Every point of the draw, for each launch shape: who can be drawn is exactly
// the eligible set, so no draw value reaches a member the rule excludes.
func TestChooseReviewerDrawsOnlyEligibleMembers(t *testing.T) {
	entries := signerEntries(t)
	cases := []struct {
		name string
		c    ReviewerChoice
		want []string
	}{
		{"first", ReviewerChoice{Slot: FirstSigner}, []string{"nan/mimo-v2.6-flash", "nan/deepseek-v4-flash"}},
		{"fallback", ReviewerChoice{Slot: FirstSigner, FallbackReason: "quota"}, []string{"anthropic-review/claude-haiku-5-5"}},
		{"second after xiaomi", ReviewerChoice{Slot: SecondSigner, FirstVendor: "xiaomi"},
			[]string{"anthropic-review/claude-sonnet-5-5", "nan/glm5.3-flash"}},
		{"second after zhipu", ReviewerChoice{Slot: SecondSigner, FirstVendor: "zhipu"}, []string{"anthropic-review/claude-sonnet-5-5"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got []string
			for i := 0; ; i++ {
				e, err := ChooseReviewer(entries, c.c, func(n int) int {
					if i >= n {
						return -1
					}
					return i
				})
				if err != nil {
					break
				}
				got = append(got, e.ID)
			}
			if strings.Join(got, ",") != strings.Join(c.want, ",") {
				t.Fatalf("drawable = %v, want %v", got, c.want)
			}
		})
	}
}

// --reviewer cannot reach a member the draw would never pick.
func TestChooseReviewerRefusesANamedIneligibleMember(t *testing.T) {
	entries := signerEntries(t)
	never := func(int) int { t.Fatal("a named member must not draw"); return 0 }
	cases := []struct {
		name string
		c    ReviewerChoice
		want string
	}{
		{"second-only as first", ReviewerChoice{Slot: FirstSigner, Want: "anthropic-review/claude-sonnet-5-5"}, "--second"},
		{"fallback without reason", ReviewerChoice{Slot: FirstSigner, Want: "anthropic-review/claude-haiku-5-5"}, "--fallback-reason"},
		{"first signer with a reason", ReviewerChoice{Slot: FirstSigner, Want: "nan/mimo-v2.6-flash", FallbackReason: "timeout"}, "signs first"},
		{"unknown reason", ReviewerChoice{Slot: FirstSigner, Want: "anthropic-review/claude-haiku-5-5", FallbackReason: "slow"}, "not a classified"},
		{"reason on the second slot", ReviewerChoice{Slot: SecondSigner, FallbackReason: "timeout"}, "no fallback"},
		{"first signer as second", ReviewerChoice{Slot: SecondSigner, Want: "nan/deepseek-v4-flash", FirstVendor: "xiaomi"}, "signs: second"},
		{"same vendor as second", ReviewerChoice{Slot: SecondSigner, Want: "nan/glm5.3-flash", FirstVendor: "zhipu"}, "same as the first signer"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := ChooseReviewer(entries, c.c, never)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("want a refusal naming %q, got: %v", c.want, err)
			}
		})
	}
	if e, err := ChooseReviewer(entries, ReviewerChoice{Slot: FirstSigner, Want: "anthropic-review/claude-haiku-5-5", FallbackReason: "rate-limit"}, never); err != nil || !strings.Contains(e.ID, "haiku") {
		t.Fatalf("a named fallback with a reason must resolve, got %v %v", e, err)
	}
}

func TestSecondSlotPromptAndSession(t *testing.T) {
	p := ReviewPromptIn(SecondSigner, "AI-001-x", "/r", "anthropic-review/claude-sonnet-5-5", "pi", "/s", "")
	for _, want := range []string{"specs/AI-001-x/review-second.md", "SECOND signer", "Do NOT read"} {
		if !strings.Contains(p, want) {
			t.Errorf("second-slot prompt must say %q", want)
		}
	}
	if strings.Contains(ReviewPrompt("AI-001-x", "/r", "nan/x", "pi", "/s", ""), "SECOND signer") {
		t.Error("the first-slot prompt must not carry the second-signer brief")
	}
	if SecondSigner.Session("AI-001-x") == FirstSigner.Session("AI-001-x") {
		t.Error("the two slots must run under different tmux sessions")
	}
}

// The shipped pool under amendment B: it loads (so no Anthropic member signs
// first), every member names its vendor, and a `risk: high` spec has a second
// signer of another vendor than every first signer.
func TestShippedPoolCanSignAHighRiskSpec(t *testing.T) {
	entries, err := LoadReviewerPoolEntries(shippedRepoRoot(t))
	if err != nil {
		t.Fatalf("the shipped pool must load under amendment B: %v", err)
	}
	for _, e := range entries {
		if strings.TrimSpace(e.Vendor) == "" {
			t.Errorf("pool entry %q declares no vendor; the second-signature check compares vendors", e.ID)
		}
		if e.SignatureRole() != SignsFirst {
			continue
		}
		if _, err := ChooseReviewer(entries, ReviewerChoice{Slot: SecondSigner, FirstVendor: e.vendor()}, func(int) int { return 0 }); err != nil {
			t.Errorf("after first signer %q, no second signer is eligible: %v", e.ID, err)
		}
	}
}
