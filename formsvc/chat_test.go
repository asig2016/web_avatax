package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// fakeRocketChat answers /api/v1/livechat/config like Rocket.Chat: department AVATAX (id d1)
// is online when deptOnline is true; the global status is always online.
func fakeRocketChat(t *testing.T, deptOnline *bool, calls *int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/livechat/config" {
			http.NotFound(w, r)
			return
		}
		*calls++
		online := true
		if r.URL.Query().Get("department") == "d1" {
			online = *deptOnline
		}
		json.NewEncoder(w).Encode(map[string]any{"config": map[string]any{
			"enabled": true, "online": online,
			"departments": []map[string]string{{"_id": "d1", "name": "AVATAX"}, {"_id": "d2", "name": "Other"}},
		}})
	}))
}

func chatGet(s *server) chatResult {
	rec := httptest.NewRecorder()
	s.routes().ServeHTTP(rec, httptest.NewRequest("GET", "/api/chat-status", nil))
	var res chatResult
	json.NewDecoder(rec.Body).Decode(&res)
	return res
}

func TestChatStatusDepartmentAndCache(t *testing.T) {
	online, calls := false, 0
	rc := fakeRocketChat(t, &online, &calls)
	defer rc.Close()

	s, _ := testServer()
	s.chat = newChatStatus(rc.URL+"/", "AVATAX")
	now := t0
	s.chat.now = func() time.Time { return now }

	if res := chatGet(s); res.Online {
		t.Fatalf("department offline must give online=false, got %+v", res)
	}
	online = true
	if res := chatGet(s); res.Online || calls != 2 {
		t.Fatalf("result must be cached for %s (calls=%d, %+v)", chatCacheTTL, calls, res)
	}
	now = now.Add(chatCacheTTL + time.Second)
	if res := chatGet(s); !res.Online || res.Department != "d1" {
		t.Fatalf("want online with department d1 after cache expiry, got %+v", res)
	}
}

func TestChatStatusUnknownDepartmentOrDisabled(t *testing.T) {
	online, calls := true, 0
	rc := fakeRocketChat(t, &online, &calls)
	defer rc.Close()

	s, _ := testServer()
	s.chat = newChatStatus(rc.URL, "Missing")
	if res := chatGet(s); res.Online {
		t.Fatalf("unknown department must give online=false, got %+v", res)
	}
	s.chat = newChatStatus("", "AVATAX")
	if res := chatGet(s); res.Online {
		t.Fatalf("no CHAT_URL must give online=false, got %+v", res)
	}
	s.chat = newChatStatus("http://127.0.0.1:1", "AVATAX")
	if res := chatGet(s); res.Online {
		t.Fatalf("unreachable server must give online=false, got %+v", res)
	}
}
