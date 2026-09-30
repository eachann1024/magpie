package gateway

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/catalog"
	"github.com/yetone/magpie/internal/provider"
)

func TestDeclaredGroupImageDispatch(t *testing.T) {
	cases := []struct {
		name                       string
		input, childInput, members []string
		child, rule                bool
		status                     int
		model                      string
	}{
		{name: "automatic mixed", members: []string{"qa/text", "qa/vision"}, status: 400},
		{name: "declared mixed", input: []string{"text", "image"}, members: []string{"qa/text", "qa/vision"}, status: 200, model: "vision"},
		{name: "declared text vision", input: []string{"text"}, members: []string{"qa/vision"}, status: 400},
		{name: "nested image", child: true, childInput: []string{"text", "image"}, members: []string{"qa/text", "qa/vision"}, status: 200, model: "vision"},
		{name: "nested text", child: true, childInput: []string{"text"}, members: []string{"qa/vision"}, status: 400},
		{name: "image parent text child", input: []string{"text", "image"}, child: true, childInput: []string{"text"}, members: []string{"qa/vision"}, status: 400},
		{name: "all text", input: []string{"text", "image"}, members: []string{"qa/text"}, status: 400},
		{name: "unknown", input: []string{"text", "image"}, members: []string{"qa/unknown"}, status: 200, model: "unknown"},
		{name: "rule image child", child: true, rule: true, childInput: []string{"text", "image"}, members: []string{"qa/text", "qa/vision"}, status: 200, model: "vision"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fresh(t)
			noVision(t)
			var sent []string
			up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				b, _ := io.ReadAll(r.Body)
				var req struct{ Model string }
				json.Unmarshal(b, &req)
				sent = append(sent, req.Model)
				if req.Model == "text" {
					t.Error("native image sent to text member")
				}
				w.Header().Set("Content-Type", "application/json")
				io.WriteString(w, `{"id":"x","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]}`)
			}))
			defer up.Close()
			if err := provider.Save(provider.Provider{ID: "qa", Name: "QA", Chat: up.URL + "/v1", Key: "fixture", Models: []string{"text", "vision", "unknown"}}); err != nil {
				t.Fatal(err)
			}
			if err := catalog.SaveLive("qa", up.URL+"/v1", []catalog.Model{{ID: "text", ImageInput: imageInputBool(false)}, {ID: "vision", Images: true, ImageInput: imageInputBool(true)}, {ID: "unknown"}}); err != nil {
				t.Fatal(err)
			}
			root := provider.Group{ID: "root", Members: tc.members, Input: tc.input, Routing: provider.Ordered}
			if tc.child {
				if err := provider.SaveGroup(provider.Group{ID: "child", Members: tc.members, Input: tc.childInput, Routing: provider.Ordered}); err != nil {
					t.Fatal(err)
				}
				root.Members = []string{"group/child"}
				if tc.rule {
					root.Members = append(root.Members, "qa/text")
					root.Rules = []provider.Rule{{Use: "group/child", Images: true}}
				}
			}
			if err := provider.SaveGroup(root); err != nil {
				t.Fatal(err)
			}
			code, body := postAs(t, New(), "", `{"model":"group/root","messages":[{"role":"user","content":[{"type":"text","text":"read"},{"type":"image_url","image_url":{"url":"data:image/png;base64,aGVsbG8="}}]}]}`)
			if code != tc.status {
				t.Fatalf("%d %s; sent %v", code, body, sent)
			}
			if tc.model != "" && (len(sent) != 1 || sent[0] != tc.model) {
				t.Fatalf("wrong member: %v", sent)
			}
			if tc.status == 400 && len(sent) != 0 {
				t.Fatalf("rejected image reached upstream: %v", sent)
			}
		})
	}
}

func TestDeclaredTextGroupOmitsHistoricalImages(t *testing.T) {
	fresh(t)
	noVision(t)
	var sent string
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		sent = string(b)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"x","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]}`)
	}))
	defer up.Close()
	if err := provider.Save(provider.Provider{ID: "qa", Name: "QA", Chat: up.URL + "/v1", Key: "fixture", Models: []string{"vision"}}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.SaveLive("qa", up.URL+"/v1", []catalog.Model{{ID: "vision", Images: true, ImageInput: imageInputBool(true)}}); err != nil {
		t.Fatal(err)
	}
	if err := provider.SaveGroup(provider.Group{ID: "text", Members: []string{"qa/vision"}, Input: []string{"text"}}); err != nil {
		t.Fatal(err)
	}
	code, body := postAs(t, New(), "", `{"model":"group/text","messages":[{"role":"user","content":[{"type":"image_url","image_url":{"url":"data:image/png;base64,aGVsbG8="}}]},{"role":"assistant","content":"old"},{"role":"user","content":"continue"}]}`)
	if code != 200 || strings.Contains(sent, "image_url") {
		t.Fatalf("%d %s; upstream %s", code, body, sent)
	}
}
