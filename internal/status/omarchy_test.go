package status

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/omarchy"
	"github.com/fynxlabs/rwr/internal/types"
)

func TestOmarchyRowsReuseDiscovery(t *testing.T) {
	t.Parallel()
	for _, unavailable := range []bool{false, true} {
		t.Run(map[bool]string{false: "available", true: "unavailable"}[unavailable], func(t *testing.T) {
			t.Parallel()
			home := t.TempDir()
			if err := os.MkdirAll(filepath.Join(home, ".config/omarchy"), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(home, ".config/omarchy/shell.json"), []byte(`{"bar":{},"idle":{"lock":300},"plugins":[]}`), 0600); err != nil {
				t.Fatal(err)
			}
			catalogs, lists := 0, 0
			client := &omarchy.Client{Home: home, LookPath: func(name string) (string, error) { return name, nil }, Read: func(_ context.Context, _ string, args ...string) ([]byte, error) {
				switch strings.Join(args, " ") {
				case "plugin catalog":
					catalogs++
					if unavailable {
						return nil, errors.New("offline")
					}
					return []byte(`[{"id":"omarchy.idle","sourceDir":"/stock","kinds":["service"],"firstParty":true}]`), nil
				case "plugin list --json":
					lists++
					return []byte(`[{"id":"omarchy.idle","enabled":true}]`), nil
				default:
					t.Fatalf("unexpected query: %v", args)
					return nil, nil
				}
			}}
			plan := &types.Plan{}
			for _, value := range []int{300, 600, 300} {
				raw, err := json.Marshal(omarchy.Operation{ID: "shell/idle/lock", Kind: "setting", Path: []string{"idle", "lock"}, Value: value})
				if err != nil {
					t.Fatal(err)
				}
				plan.Resources = append(plan.Resources, types.Resource{Processor: types.BlueprintTypeOmarchy, DesiredState: raw})
			}
			q := NewQuerier()
			q.omarchyClient = client
			rows := Rows(plan, nil, q)
			want := []Class{InSync, ModifiedItem, InSync}
			for i, row := range rows {
				if unavailable {
					want[i] = UnknownItem
				}
				if row.Class != want[i] {
					t.Fatalf("row %d = %+v, want %s", i, row, want[i])
				}
			}
			wantLists := 1
			if unavailable {
				wantLists = 0
			}
			if catalogs != 1 || lists != wantLists {
				t.Fatalf("queries: catalog=%d list=%d", catalogs, lists)
			}
			// A separate status run gets a new observation, including after failure.
			q = NewQuerier()
			q.omarchyClient = client
			Rows(plan, nil, q)
			if catalogs != 2 {
				t.Fatalf("new run reused old snapshot: %d", catalogs)
			}
		})
	}
}
