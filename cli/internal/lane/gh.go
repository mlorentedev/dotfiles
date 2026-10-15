package lane

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// GH reads issues through the gh CLI, in the repository of the working
// directory it runs in.
type GH struct{ Run Runner }

// Issue implements IssueSource.
func (g GH) Issue(number int) (Issue, error) {
	out, err := g.Run("gh", "issue", "view", strconv.Itoa(number), "--json", "number,title,state,assignees")
	if err != nil {
		return Issue{}, err
	}
	var raw struct {
		Number    int    `json:"number"`
		Title     string `json:"title"`
		State     string `json:"state"`
		Assignees []struct {
			Login string `json:"login"`
		} `json:"assignees"`
	}
	if err := json.Unmarshal([]byte(out), &raw); err != nil {
		return Issue{}, fmt.Errorf("gh issue view %d: %w", number, err)
	}
	is := Issue{Number: raw.Number, Title: raw.Title, State: raw.State}
	for _, a := range raw.Assignees {
		is.Assignees = append(is.Assignees, a.Login)
	}
	return is, nil
}
