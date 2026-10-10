package library

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstructionSourceAliasDoesNotGrow(t *testing.T) {
	for _, kind := range []string{"symlink", "hardlink"} {
		t.Run(kind, func(t *testing.T) {
			h := sandbox(t)
			source := filepath.Join(h, "dotfiles", "AGENTS.md")
			original := "Use tabs.\n"
			write(t, source, original)
			linkIn(t, source, setPath(defaultSet))
			cx := filepath.Join(h, ".codex", "AGENTS.md")
			if kind == "symlink" {
				linkIn(t, source, cx)
			} else if err := os.Link(source, cx); err != nil {
				t.Skip(err)
			}
			check := func(r *Result, err error) {
				t.Helper()
				if err != nil {
					t.Fatal(err)
				}
				if len(r.Problems) != 0 || len(r.Changed) != 0 {
					t.Fatalf("result: %+v", r)
				}
				if got := read(t, source); got != original {
					t.Fatalf("source changed: %q", got)
				}
				if got := read(t, cx); got != original {
					t.Fatalf("agent changed: %q", got)
				}
			}
			check(SaveInstructions(InstructionsChange{Agents: []string{"codex"}}))
			// Sync reloads the library from disk, as each app restart does.
			for range 5 {
				check(Sync())
			}
			updated := "Use spaces."
			original = updated + "\n"
			check(SaveInstructions(InstructionsChange{Shared: &updated}))
			check(Sync())
			extra := "Codex only."
			r, err := SaveInstructions(InstructionsChange{Extra: map[string]*string{"codex": &extra}})
			if err != nil || len(r.Problems) != 1 || !strings.Contains(r.Problems[0].Error, "instruction source") {
				t.Fatalf("extra: %+v %v", r, err)
			}
			if read(t, source) != original {
				t.Fatal("extra changed source")
			}
			check(SaveInstructions(InstructionsChange{Agents: []string{}}))
			if _, err := ImportInstructions("codex"); err == nil || !strings.Contains(err.Error(), "instruction source") {
				t.Fatalf("import: %v", err)
			}
			stillLink(t, setPath(defaultSet))
			if kind == "symlink" {
				stillLink(t, cx)
			} else {
				a, _ := os.Stat(source)
				b, _ := os.Stat(cx)
				if !os.SameFile(a, b) {
					t.Fatal("hard link split")
				}
			}
		})
	}
}

func TestInstructionInactiveAndExtraSourcesStayUntouched(t *testing.T) {
	for _, kind := range []string{"inactive", "extra"} {
		t.Run(kind, func(t *testing.T) {
			h := sandbox(t)
			source := setPath("other")
			if kind == "extra" {
				source = extraPath("claude")
			}
			write(t, source, "Source only.\n")
			cx := filepath.Join(h, ".codex", "AGENTS.md")
			linkIn(t, source, cx)
			shared := "Shared."
			change := InstructionsChange{Shared: &shared, Agents: []string{"codex"}}
			if kind == "inactive" {
				change.Create = &InstrSet{ID: "other", Name: "Other"}
			}
			r, err := SaveInstructions(change)
			if err != nil {
				t.Fatal(err)
			}
			if len(r.Problems) != 1 {
				t.Fatalf("result: %+v", r)
			}
			if got := read(t, source); got != "Source only.\n" {
				t.Fatalf("source changed: %q", got)
			}
		})
	}
}

func TestNestedInstructionBlock(t *testing.T) {
	nested := "Before.\n\n" + blockBegin + "\nOld.\n" + blockBegin + "\nOld.\n" + blockEnd + "\n" + blockEnd + "\n\nAfter.\n"
	before, _, after, found := split(nested)
	if !found || before != "Before.\n\n" || after != "\n\nAfter.\n" {
		t.Fatalf("split: %q %q %v", before, after, found)
	}
	want := "Before.\n\n" + blockBegin + "\nNew.\n" + blockEnd + "\n\nAfter.\n"
	if got := withBlock(nested, "New."); got != want {
		t.Fatalf("rewrite: %q", got)
	}
	if got := own(nested); got != "Before.\n\nAfter." {
		t.Fatalf("own: %q", got)
	}
	malformed := blockBegin + "\n" + blockBegin + "\nText.\n" + blockEnd
	if _, _, _, found := split(malformed); found {
		t.Fatal("unbalanced block parsed")
	}
}

func TestOrphanInstructionEndsAreCleaned(t *testing.T) {
	h := sandbox(t)
	shared := "Use tabs."
	ok(t)(SaveInstructions(InstructionsChange{Shared: &shared, Agents: []string{"codex"}}))
	cx := filepath.Join(h, ".codex", "AGENTS.md")
	clean := read(t, cx)
	write(t, cx, clean+"\n"+strings.Repeat(blockEnd+"\n\n", 10000)+"After.\n")
	ok(t)(Sync())
	want := clean + "\nAfter.\n"
	if got := read(t, cx); got != want {
		t.Fatalf("orphan ends survived: %d bytes", len(got))
	}
	for range 5 {
		r := ok(t)(Sync())
		if len(r.Changed) != 0 || read(t, cx) != want {
			t.Fatal("sync was not stable")
		}
	}
}

func TestInstructionSourceAliasKeepsMCPAndSkillsWorking(t *testing.T) {
	h := sandbox(t)
	source := filepath.Join(h, "icloud", "AGENTS.md")
	write(t, source, "Use tabs.\n")
	linkIn(t, source, setPath(defaultSet))
	cx := filepath.Join(h, ".codex", "AGENTS.md")
	linkIn(t, source, cx)
	config := filepath.Join(h, "icloud", "config.toml")
	write(t, config, "model = \"my-model\"\n")
	if err := os.Remove(filepath.Join(h, ".codex", "config.toml")); err != nil {
		t.Fatal(err)
	}
	linkIn(t, config, filepath.Join(h, ".codex", "config.toml"))
	ok(t)(SaveInstructions(InstructionsChange{Agents: []string{"codex"}}))
	server := Server{Name: "test", Transport: "stdio", Command: "test-server", Agents: []string{"codex"}}
	ok(t)(SaveServer("", server))
	src := filepath.Join(h, "src", "skills")
	skill(t, filepath.Join(src, "test-skill"), "test-skill", "Test skill")
	ok(t)(InstallSkills(src, []string{"test-skill"}, []string{"codex"}))
	ok(t)(Sync())
	servers, err := targetByID("codex").MCP.read()
	if err != nil || servers["test"] == nil || !servers["test"].same(&server) {
		t.Fatalf("MCP: %+v %v", servers, err)
	}
	if got := read(t, filepath.Join(h, ".codex", "skills", "test-skill", "SKILL.md")); !strings.Contains(got, "Test skill") {
		t.Fatalf("skill: %q", got)
	}
	if read(t, source) != "Use tabs.\n" {
		t.Fatal("instructions source changed")
	}
	if !strings.Contains(read(t, config), "my-model") {
		t.Fatal("model setting changed")
	}
	stillLink(t, cx)
	stillLink(t, setPath(defaultSet))
	stillLink(t, filepath.Join(h, ".codex", "config.toml"))
}
