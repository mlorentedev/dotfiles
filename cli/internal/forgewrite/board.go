package forgewrite

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

// Board names a GitHub Project (v2) that issues are placed on. It is declared
// per host in forge/hosts.json, never guessed.
type Board struct {
	Owner  string `json:"owner"`
	Number int    `json:"number"`
}

// BoardFields are the values to set on the board item. Status, Priority and
// Type are single-select options matched by name; ID is free text.
type BoardFields struct {
	Status, Priority, Type, ID string
}

// BoardPlan is a board placement resolved against the live board but not yet
// written. Resolving first lets a bad option name fail before the issue exists.
type BoardPlan struct {
	Board     Board
	ProjectID string
	Edits     []BoardEdit
}

// BoardEdit is one field write: an option id for a single-select field, or
// text for a text field.
type BoardEdit struct {
	FieldID  string
	OptionID string
	Text     string
}

// BoardWriter is the optional capability of a backend whose forge has Projects.
// It is not part of Backend: a Gitea origin does not satisfy it, and dotf forge
// says so instead of dropping the flag.
type BoardWriter interface {
	// ResolveBoard looks the project and every named field up and writes nothing.
	ResolveBoard(ctx context.Context, b Board, f BoardFields) (BoardPlan, error)
	// PlaceOnBoard adds the issue at url to the board and sets the planned fields.
	PlaceOnBoard(ctx context.Context, plan BoardPlan, url string) error
}

// LoadBoards reads the "boards" map of forge/hosts.json: host to project. A
// missing file or key is an empty map.
func LoadBoards(path string) (map[string]Board, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]Board{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var f struct {
		Boards map[string]Board `json:"boards"`
	}
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	for host, b := range f.Boards {
		if b.Owner == "" || b.Number < 1 {
			return nil, fmt.Errorf("ERROR: board for %q in %s is incomplete. WHY: a board needs an owner and a project number. FIX: set \"owner\" and \"number\"", host, path)
		}
	}
	if f.Boards == nil {
		return map[string]Board{}, nil
	}
	return f.Boards, nil
}

// ResolveBoard implements BoardWriter. Field and option ids are read from the
// live board each time, never pasted, so a renamed option fails loudly.
func (g GitHub) ResolveBoard(ctx context.Context, b Board, f BoardFields) (BoardPlan, error) {
	num := strconv.Itoa(b.Number)
	var view struct {
		ID string `json:"id"`
	}
	if err := g.ghJSON(ctx, &view, "project", "view", num, "--owner", b.Owner, "--format", "json"); err != nil {
		return BoardPlan{}, err
	}
	var list struct {
		Fields []boardField `json:"fields"`
	}
	if err := g.ghJSON(ctx, &list, "project", "field-list", num, "--owner", b.Owner, "--format", "json", "--limit", "100"); err != nil {
		return BoardPlan{}, err
	}
	if view.ID == "" {
		return BoardPlan{}, fmt.Errorf("ERROR: project %d of %s has no id. WHY: gh returned none. FIX: check the board in forge/hosts.json", b.Number, b.Owner)
	}
	plan := BoardPlan{Board: b, ProjectID: view.ID}
	selects := []struct{ field, value string }{{"Status", f.Status}, {"Priority", f.Priority}, {"Type", f.Type}}
	for _, s := range selects {
		edit, err := selectEdit(list.Fields, s.field, s.value, b)
		if err != nil {
			return BoardPlan{}, err
		}
		plan.Edits = append(plan.Edits, edit)
	}
	if f.ID != "" {
		for _, fl := range list.Fields {
			if fl.Name == "ID" {
				plan.Edits = append(plan.Edits, BoardEdit{FieldID: fl.ID, Text: f.ID})
			}
		}
		if len(plan.Edits) == len(selects) {
			return BoardPlan{}, fmt.Errorf("ERROR: the board has no ID field. WHY: --id sets a text field named ID on project %d of %s. FIX: drop --id, or add the field", b.Number, b.Owner)
		}
	}
	return plan, nil
}

// boardField is one field of `gh project field-list`; Options is set for a
// single-select field only.
type boardField struct {
	ID      string        `json:"id"`
	Name    string        `json:"name"`
	Options []boardOption `json:"options"`
}

type boardOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func selectEdit(fields []boardField, field, value string, b Board) (BoardEdit, error) {
	for _, fl := range fields {
		if fl.Name != field {
			continue
		}
		for _, o := range fl.Options {
			if o.Name == value {
				return BoardEdit{FieldID: fl.ID, OptionID: o.ID}, nil
			}
		}
		names := make([]string, 0, len(fl.Options))
		for _, o := range fl.Options {
			names = append(names, o.Name)
		}
		return BoardEdit{}, fmt.Errorf("ERROR: %s %q is not an option of project %d of %s. WHY: the board offers %v. FIX: pick one of those", field, value, b.Number, b.Owner, names)
	}
	return BoardEdit{}, fmt.Errorf("ERROR: the board has no %s field. WHY: project %d of %s does not define it. FIX: check forge/hosts.json points at the right project", field, b.Number, b.Owner)
}

// PlaceOnBoard implements BoardWriter. GitHub's addProjectV2ItemById returns
// the existing card when the issue is already on the board, so adding it again
// after a failed field write does not create a second card.
func (g GitHub) PlaceOnBoard(ctx context.Context, plan BoardPlan, url string) error {
	num := strconv.Itoa(plan.Board.Number)
	var item struct {
		ID string `json:"id"`
	}
	if err := g.ghJSON(ctx, &item, "project", "item-add", num, "--owner", plan.Board.Owner, "--url", url, "--format", "json"); err != nil {
		return err
	}
	if item.ID == "" {
		return fmt.Errorf("gh project item-add returned no item id for %s", url)
	}
	for i, e := range plan.Edits {
		args := []string{"project", "item-edit", "--id", item.ID, "--project-id", plan.ProjectID, "--field-id", e.FieldID}
		if e.OptionID != "" {
			args = append(args, "--single-select-option-id", e.OptionID)
		} else {
			args = append(args, "--text", e.Text)
		}
		if _, err := g.Run(ctx, nil, args...); err != nil {
			return fmt.Errorf("the card %s was added but only %d of %d fields were set (gh project item-edit: %w)", item.ID, i, len(plan.Edits), err)
		}
	}
	return nil
}

// ghJSON runs gh and decodes its JSON stdout into out.
func (g GitHub) ghJSON(ctx context.Context, out any, args ...string) error {
	raw, err := g.Run(ctx, nil, args...)
	if err != nil {
		return fmt.Errorf("gh %s %s: %w", args[0], args[1], err)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("gh %s %s answered something unreadable", args[0], args[1])
	}
	return nil
}
