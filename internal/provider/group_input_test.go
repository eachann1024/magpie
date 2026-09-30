package provider

import (
	"reflect"
	"testing"
)

func TestGroupInputDeclarationsAndNesting(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("PATH", "")
	yes, no := true, false
	p := Provider{ID: "qa", Name: "QA", Key: "fixture", Chat: "http://127.0.0.1:1/v1"}
	entries := []Entry{{ID: "qa/text", Model: "text", Provider: p, ImageInput: &no}, {ID: "qa/vision", Model: "vision", Provider: p, Images: true, ImageInput: &yes}}
	for _, input := range [][]string{nil, {"text"}, {"text", "image"}} {
		g := Group{ID: "child", Members: []string{"qa/text", "qa/vision"}, Input: input, Family: "kept", Context: 123456}
		if err := SaveGroup(g); err != nil {
			t.Fatal(err)
		}
		if err := SaveGroup(Group{ID: "parent", Members: []string{"group/child"}}); err != nil {
			t.Fatal(err)
		}
		got := map[string]Entry{}
		for _, e := range groupEntries(entries) {
			got[e.Group] = e
		}
		want := reflect.DeepEqual(input, []string{"text", "image"})
		if got["child"].Images != want || got["parent"].Images != want {
			t.Fatalf("input %v: child %+v parent %+v", input, got["child"], got["parent"])
		}
		if !reflect.DeepEqual(got["child"].Input, input) {
			t.Fatalf("declaration lost: %+v", got["child"])
		}
	}
	if err := RenameGroup("child", "renamed"); err != nil {
		t.Fatal(err)
	}
	for _, g := range load().Groups {
		if g.ID == "renamed" && (g.Family != "kept" || g.Context != 123456 || !reflect.DeepEqual(g.Input, []string{"text", "image"})) {
			t.Fatalf("rename lost settings: %+v", g)
		}
	}
	for _, bad := range [][]string{{}, {"image"}, {"text", "audio"}, {"text", "pdf"}} {
		if err := SaveGroup(Group{ID: "bad", Members: []string{"qa/vision"}, Input: bad}); err == nil {
			t.Fatalf("accepted %v", bad)
		}
	}
	if err := SaveGroup(Group{ID: "normalized", Members: []string{"qa/vision"}, Input: []string{" IMAGE ", "Text", "text"}}); err != nil {
		t.Fatal(err)
	}
	for _, g := range load().Groups {
		if g.ID == "normalized" && !reflect.DeepEqual(g.Input, []string{"text", "image"}) {
			t.Fatalf("not normalized: %+v", g)
		}
	}
	// Explicit text wins even over a vision request rule.
	if err := SaveGroup(Group{ID: "ruled", Members: []string{"qa/text", "qa/vision"}, Input: []string{"text"}, Rules: []Rule{{Use: "qa/vision", Images: true}}}); err != nil {
		t.Fatal(err)
	}
	for _, e := range groupEntries(entries) {
		if e.Group == "ruled" && e.Images {
			t.Fatal("rule overrode text declaration")
		}
	}
}
