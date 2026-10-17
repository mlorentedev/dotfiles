package forgewrite

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const boardView = `{"id":"PVT_board"}`

const boardFields = `{"fields":[
 {"id":"F_title","name":"Title"},
 {"id":"F_status","name":"Status","options":[{"id":"O_backlog","name":"Backlog"},{"id":"O_done","name":"Done"}]},
 {"id":"F_prio","name":"Priority","options":[{"id":"O_p1","name":"P1"},{"id":"O_p2","name":"P2"}]},
 {"id":"F_type","name":"Type","options":[{"id":"O_bug","name":"bug"},{"id":"O_chore","name":"chore"}]},
 {"id":"F_id","name":"ID"}
]}`

// ghScript answers `gh project ...` calls from a fixed script and records them.
func ghScript(t *testing.T, fields string) (GitHub, *[]string) {
	t.Helper()
	var calls []string
	g := GitHub{Run: func(_ context.Context, stdin []byte, args ...string) ([]byte, error) {
		calls = append(calls, strings.Join(args, " "))
		if len(stdin) != 0 {
			t.Errorf("a board call carried stdin: %q", stdin)
		}
		switch args[1] {
		case "view":
			return []byte(boardView), nil
		case "field-list":
			return []byte(fields), nil
		case "item-add":
			return []byte(`{"id":"ITEM_1"}`), nil
		}
		return []byte(`{}`), nil
	}}
	return g, &calls
}

var testBoard = Board{Owner: "mlorentedev", Number: 1}

func TestResolveBoardMapsNamesToIDsAndWritesNothing(t *testing.T) {
	g, calls := ghScript(t, boardFields)
	plan, err := g.ResolveBoard(context.Background(), testBoard, BoardFields{Status: "Backlog", Priority: "P2", Type: "chore", ID: "TOOL-031-slug"})
	if err != nil {
		t.Fatal(err)
	}
	want := []BoardEdit{{FieldID: "F_status", OptionID: "O_backlog"}, {FieldID: "F_prio", OptionID: "O_p2"}, {FieldID: "F_type", OptionID: "O_chore"}, {FieldID: "F_id", Text: "TOOL-031-slug"}}
	if plan.ProjectID != "PVT_board" || len(plan.Edits) != len(want) {
		t.Fatalf("plan = %+v", plan)
	}
	for i, e := range want {
		if plan.Edits[i] != e {
			t.Errorf("edit %d = %+v, want %+v", i, plan.Edits[i], e)
		}
	}
	for _, c := range *calls {
		if strings.Contains(c, "item-add") || strings.Contains(c, "item-edit") {
			t.Errorf("resolving wrote to the board: %s", c)
		}
	}
}

func TestResolveBoardNamesTheValidOptionsOnAMiss(t *testing.T) {
	g, _ := ghScript(t, boardFields)
	_, err := g.ResolveBoard(context.Background(), testBoard, BoardFields{Status: "Backlog", Priority: "P9", Type: "chore"})
	if err == nil || !strings.Contains(err.Error(), "P9") || !strings.Contains(err.Error(), "P1") || !strings.Contains(err.Error(), "FIX:") {
		t.Fatalf("want ERROR/WHY/FIX naming the options, got %v", err)
	}
}

func TestResolveBoardRefusesIDWhenTheBoardHasNone(t *testing.T) {
	noID := strings.Replace(boardFields, `,
 {"id":"F_id","name":"ID"}`, "", 1)
	g, _ := ghScript(t, noID)
	if _, err := g.ResolveBoard(context.Background(), testBoard, BoardFields{Status: "Backlog", Priority: "P2", Type: "bug", ID: "X-1"}); err == nil {
		t.Fatal("--id with no ID field must be refused, not dropped")
	}
	if _, err := g.ResolveBoard(context.Background(), testBoard, BoardFields{Status: "Backlog", Priority: "P2", Type: "bug"}); err != nil {
		t.Errorf("no --id needs no ID field, got %v", err)
	}
}

func TestPlaceOnBoardAddsTheCardThenSetsEachField(t *testing.T) {
	g, calls := ghScript(t, boardFields)
	plan := BoardPlan{Board: testBoard, ProjectID: "PVT_board", Edits: []BoardEdit{{FieldID: "F_status", OptionID: "O_backlog"}, {FieldID: "F_id", Text: "T-1"}}}
	if err := g.PlaceOnBoard(context.Background(), plan, "https://github.com/o/n/issues/4"); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"project item-add 1 --owner mlorentedev --url https://github.com/o/n/issues/4 --format json",
		"project item-edit --id ITEM_1 --project-id PVT_board --field-id F_status --single-select-option-id O_backlog",
		"project item-edit --id ITEM_1 --project-id PVT_board --field-id F_id --text T-1",
	}
	if strings.Join(*calls, "\n") != strings.Join(want, "\n") {
		t.Errorf("gh calls:\n%s\nwant:\n%s", strings.Join(*calls, "\n"), strings.Join(want, "\n"))
	}
}

func TestPlaceOnBoardReportsAFailedFieldWrite(t *testing.T) {
	g := GitHub{Run: func(_ context.Context, _ []byte, args ...string) ([]byte, error) {
		if args[1] == "item-add" {
			return []byte(`{"id":"ITEM_1"}`), nil
		}
		return nil, errors.New("boom")
	}}
	err := g.PlaceOnBoard(context.Background(), BoardPlan{Board: testBoard, ProjectID: "P", Edits: []BoardEdit{{FieldID: "F", OptionID: "O"}}}, "u")
	if err == nil || !strings.Contains(err.Error(), "boom") || !strings.Contains(err.Error(), "ITEM_1") || !strings.Contains(err.Error(), "0 of 1") {
		t.Fatalf("got %v", err)
	}
}

func TestLoadBoards(t *testing.T) {
	dir := t.TempDir()
	if got, err := LoadBoards(filepath.Join(dir, "absent.json")); err != nil || len(got) != 0 {
		t.Errorf("a missing file is an empty map, got %v, %v", got, err)
	}
	ok := filepath.Join(dir, "ok.json")
	if err := os.WriteFile(ok, []byte(`{"hosts":{},"boards":{"github.com":{"owner":"mlorentedev","number":1}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := LoadBoards(ok); err != nil || got["github.com"] != testBoard {
		t.Errorf("LoadBoards = %v, %v", got, err)
	}
	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte(`{"boards":{"github.com":{"owner":"x"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadBoards(bad); err == nil || !strings.Contains(err.Error(), "FIX:") {
		t.Errorf("an incomplete board must fail with ERROR/WHY/FIX, got %v", err)
	}
}

// Gitea has no Projects: it must not satisfy BoardWriter, so dotf forge can
// say so instead of dropping the flag.
func TestOnlyGitHubIsABoardWriter(t *testing.T) {
	var gh Backend = GitHub{}
	var gt Backend = Gitea{}
	if _, ok := gh.(BoardWriter); !ok {
		t.Error("GitHub must be a BoardWriter")
	}
	if _, ok := gt.(BoardWriter); ok {
		t.Error("Gitea must not be a BoardWriter")
	}
}
