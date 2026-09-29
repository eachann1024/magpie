package gui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

func TestAccountArrangeRouteAndEditorSave(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	p := provider.Provider{ID: "arrange-test", Name: "Arrange", Chat: "https://example.invalid/v1", Key: "primary", Keys: []provider.KeyAccount{{Key: "second"}}}
	if err := provider.Save(p); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	providerRoutes(mux, nil)
	order := []string{provider.KeyID("second"), provider.KeyID("primary")}
	post := func(action string, body any) *httptest.ResponseRecorder {
		b, _ := json.Marshal(body)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("POST", "/api/provider/"+action, strings.NewReader(string(b))))
		return w
	}
	w := post("arrange", map[string]any{"id": p.ID, "accountOrder": order})
	if w.Code != 200 {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	if strings.Contains(w.Body.String(), "primary") || strings.Contains(w.Body.String(), "second") {
		t.Fatal("leaked key")
	}
	// Saving an older editor form must not erase the separate arrangement.
	w = post("save", map[string]any{"id": p.ID, "name": p.Name, "chat": p.Chat})
	if w.Code != 200 {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	got, err := provider.Find(p.ID)
	if err != nil || !reflect.DeepEqual(got.AccountOrder, order) {
		t.Fatalf("lost order: %v", err)
	}
	if !reflect.DeepEqual(providerInfo(*got, nil).AccountOrder, order) {
		t.Fatal("not returned to UI")
	}
	if w = post("arrange", map[string]any{"id": p.ID, "accountOrder": []string{"missing"}}); w.Code == 200 {
		t.Fatal("accepted stale order")
	}
}
