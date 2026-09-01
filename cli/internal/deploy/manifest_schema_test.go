package deploy

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

// The version is what an old binary trips over; a field it does not know is
// invisible to it. So adding a field WITHOUT bumping ManifestVersion re-opens
// the window #1369 closed — silently, on every box running the previous
// release. This test freezes the field set each version was declared for: a
// new json tag on Config or Manifest fails here until the version moves and
// the frozen set is extended, which is the moment to ask whether old readers
// would misread the new field (strategy did; a purely additive field might
// not, and then the version still moves — the cost is one line).
func TestManifestVersion_FreezesTheFieldSet(t *testing.T) {
	frozen := map[int][]string{
		2: {"$comment", "configs", "dst", "mode", "name", "render", "requires", "src", "strategy", "version"},
		3: {"$comment", "configs", "dst", "mode", "name", "paths", "render", "requires", "src", "strategy", "version"},
	}
	want, ok := frozen[ManifestVersion]
	if !ok {
		t.Fatalf("ManifestVersion %d has no frozen field set here — add one, and say what changed", ManifestVersion)
	}
	var got []string
	for _, typ := range []reflect.Type{reflect.TypeOf(Manifest{}), reflect.TypeOf(Config{})} {
		for i := 0; i < typ.NumField(); i++ {
			tag := typ.Field(i).Tag.Get("json")
			if name := strings.Split(tag, ",")[0]; name != "" && name != "-" {
				got = append(got, name)
			}
		}
	}
	sort.Strings(got)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("the manifest schema changed without a version bump:\n got %v\nwant %v (ManifestVersion %d)\nbump ManifestVersion and ai/deploy.json together, then extend the frozen set", got, want, ManifestVersion)
	}
}
