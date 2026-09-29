package provider

import (
	"os"
	"reflect"
	"testing"
	"time"
)

func TestAccountOrderValidation(t *testing.T) {
	for _, order := range [][]string{nil, {"a"}, {"a", "a"}, {"a", "other"}, {"a", "b", "c"}} {
		if validateAccountOrder([]string{"a", "b"}, order) == nil {
			t.Fatalf("accepted %v", order)
		}
	}
	if err := validateAccountOrder([]string{"a", "b"}, []string{"b", "a"}); err != nil {
		t.Fatal(err)
	}
}

func TestAccountOrderPersistenceAllRoutingModes(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	for _, mode := range []string{"", Ordered, Rotate, LeastUsed} {
		p := Provider{ID: "arrange-test", Name: "Arrange", Chat: "https://example.invalid/v1", Key: "primary", Routing: mode,
			Keys: []KeyAccount{{Name: "Off", Key: "disabled", Off: true, Protocol: Chat}, {Name: "Extra", Key: "extra", Protocol: Anthropic}}}
		if err := Save(p); err != nil {
			t.Fatal(err)
		}
		order := []string{keyID("disabled"), keyID("extra"), keyID("primary")}
		if err := SetAccountOrder(p.ID, order); err != nil {
			t.Fatal(err)
		}
		got, err := Find(p.ID)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got.AccountOrder, order) || got.Key != p.Key || got.Routing != mode || !reflect.DeepEqual(got.Keys, p.Keys) {
			t.Fatalf("order or credentials changed: mode=%q", mode)
		}
		disk := load()
		if !reflect.DeepEqual(disk.Providers[0].AccountOrder, order) {
			t.Fatal("not persisted")
		}
		before, _ := os.ReadFile(Path())
		if err := SetAccountOrder(p.ID, []string{order[0], order[0], order[2]}); err == nil {
			t.Fatal("accepted duplicate")
		}
		after, _ := os.ReadFile(Path())
		if string(before) != string(after) {
			t.Fatal("invalid order changed the store")
		}
	}
}

func TestAccountOrderSubscriptionKeepsCredentials(t *testing.T) {
	googleSandbox(t, &fakeGoogle{})
	for _, user := range []string{"first@example.test", "second@example.test", "third@example.test"} {
		auth := googleAuth{AccessToken: "fixture", RefreshToken: "fixture-" + user, Expiry: time.Now().Add(time.Hour).UnixMilli()}
		if err := addGoogleLogin("antigravity", user, "Pro", auth); err != nil {
			t.Fatal(err)
		}
	}
	before, err := os.ReadFile(loginsPath())
	if err != nil {
		t.Fatal(err)
	}
	order := []string{"third@example.test", "first@example.test", "second@example.test"}
	for _, mode := range []string{"", Ordered, Rotate, LeastUsed} {
		if err := SetRouting("antigravity", mode); err != nil {
			t.Fatal(err)
		}
		if err := SetAccountOrder("antigravity", order); err != nil {
			t.Fatal(err)
		}
		p, err := Find("antigravity")
		if err != nil || !reflect.DeepEqual(p.AccountOrder, order) || p.Routing != mode {
			t.Fatalf("order lost: %v", err)
		}
		after, _ := os.ReadFile(loginsPath())
		if string(before) != string(after) {
			t.Fatal("arrangement changed login credentials")
		}
	}
}
