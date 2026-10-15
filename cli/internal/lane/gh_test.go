package lane

import "testing"

func TestGH_ReadsAnIssueIncludingItsAssignees(t *testing.T) {
	var seen []string
	g := GH{Run: fakeRun(map[string]string{"gh issue view 2501": `{"assignees":[{"login":"mlorentedev"}],"number":2501,"state":"OPEN","title":"BUG-143: x"}`}, &seen)}
	is, err := g.Issue(2501)
	if err != nil || is.Number != 2501 || is.State != "OPEN" || len(is.Assignees) != 1 || is.Assignees[0] != "mlorentedev" {
		t.Errorf("Issue = %+v, %v", is, err)
	}
	if seen[0] != "gh issue view 2501 --json number,title,state,assignees" {
		t.Errorf("asked %q", seen[0])
	}
}
