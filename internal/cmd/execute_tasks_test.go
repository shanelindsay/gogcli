package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/api/option"
	"google.golang.org/api/tasks/v1"
)

func TestExecute_TasksLists_JSON(t *testing.T) {
	origNew := newTasksService
	t.Cleanup(func() { newTasksService = origNew })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !(r.URL.Path == "/tasks/v1/users/@me/lists" && r.Method == http.MethodGet) {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{"id": "l1", "title": "One"},
				{"id": "l2", "title": "Two"},
			},
		})
	}))
	defer srv.Close()

	svc, err := tasks.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	newTasksService = func(context.Context, string) (*tasks.Service, error) { return svc, nil }

	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{"--output", "json", "--account", "a@b.com", "tasks", "lists", "--max", "10"}); err != nil {
				t.Fatalf("Execute: %v", err)
			}
		})
	})

	var parsed struct {
		Tasklists []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"tasklists"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("json parse: %v\nout=%q", err, out)
	}
	if len(parsed.Tasklists) != 2 || parsed.Tasklists[0].ID != "l1" || parsed.Tasklists[1].ID != "l2" {
		t.Fatalf("unexpected tasklists: %#v", parsed.Tasklists)
	}
}

func TestExecute_TasksList_JSON(t *testing.T) {
	origNew := newTasksService
	t.Cleanup(func() { newTasksService = origNew })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !(strings.HasPrefix(r.URL.Path, "/tasks/v1/lists/") && strings.HasSuffix(r.URL.Path, "/tasks") && r.Method == http.MethodGet) {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{"id": "t1", "title": "Task One", "status": "needsAction"},
				{"id": "t2", "title": "Task Two", "status": "completed"},
			},
		})
	}))
	defer srv.Close()

	svc, err := tasks.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	newTasksService = func(context.Context, string) (*tasks.Service, error) { return svc, nil }

	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{"--output", "json", "--account", "a@b.com", "tasks", "list", "l1"}); err != nil {
				t.Fatalf("Execute: %v", err)
			}
		})
	})

	var parsed struct {
		Tasks []struct {
			ID     string `json:"id"`
			Title  string `json:"title"`
			Status string `json:"status"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("json parse: %v\nout=%q", err, out)
	}
	if len(parsed.Tasks) != 2 || parsed.Tasks[0].ID != "t1" || parsed.Tasks[1].ID != "t2" {
		t.Fatalf("unexpected tasks: %#v", parsed.Tasks)
	}
}

