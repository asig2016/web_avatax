package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// chatStatus asks the Rocket.Chat server whether an agent of this site's Omnichannel
// department is available. The website calls GET /api/chat-status on page load and only
// shows the chat button when online is true; the visitor's browser itself does not contact
// the chat server before the visitor starts a chat. Results are cached for chatCacheTTL,
// so the chat server sees at most one request per TTL, independent of the number of visitors.
const chatCacheTTL = 30 * time.Second

type chatResult struct {
	Online     bool   `json:"online"`
	Department string `json:"department,omitempty"` // department id for the widget's setDepartment
}

type chatStatus struct {
	baseURL    string // e.g. https://egroupware.sigalas.eu (empty = chat disabled)
	department string // department name or id, e.g. AVATAX (empty = any agent)
	client     *http.Client
	now        func() time.Time

	mu     sync.Mutex
	until  time.Time
	result chatResult
}

func newChatStatus(baseURL, department string) *chatStatus {
	return &chatStatus{
		baseURL:    strings.TrimSuffix(strings.TrimSpace(baseURL), "/"),
		department: strings.TrimSpace(department),
		client:     &http.Client{Timeout: 5 * time.Second},
		now:        time.Now,
	}
}

type livechatConfig struct {
	Config struct {
		Enabled     bool `json:"enabled"`
		Online      bool `json:"online"`
		Departments []struct {
			ID   string `json:"_id"`
			Name string `json:"name"`
		} `json:"departments"`
	} `json:"config"`
}

func (c *chatStatus) fetchConfig(query string) (livechatConfig, error) {
	var cfg livechatConfig
	req, err := http.NewRequest("GET", c.baseURL+"/api/v1/livechat/config"+query, nil)
	if err != nil {
		return cfg, err
	}
	req.Header.Set("Accept", "application/json")
	res, err := c.client.Do(req)
	if err != nil {
		return cfg, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return cfg, fmt.Errorf("livechat config: HTTP %d", res.StatusCode)
	}
	err = json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&cfg)
	return cfg, err
}

func (c *chatStatus) fetch() (chatResult, error) {
	cfg, err := c.fetchConfig("")
	if err != nil || !cfg.Config.Enabled {
		return chatResult{}, err
	}
	if c.department == "" {
		return chatResult{Online: cfg.Config.Online}, nil
	}
	for _, d := range cfg.Config.Departments {
		if d.Name == c.department || d.ID == c.department {
			dc, err := c.fetchConfig("?department=" + url.QueryEscape(d.ID))
			if err != nil {
				return chatResult{}, err
			}
			return chatResult{Online: dc.Config.Online, Department: d.ID}, nil
		}
	}
	// Department not (yet) set up or not offered: do not show the chat for this site.
	return chatResult{}, fmt.Errorf("department %q not found", c.department)
}

func (c *chatStatus) get() chatResult {
	if c.baseURL == "" {
		return chatResult{}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.now().Before(c.until) {
		return c.result
	}
	res, err := c.fetch()
	if err != nil {
		log.Printf("chat status: %v", err)
	}
	c.result, c.until = res, c.now().Add(chatCacheTTL)
	return res
}

func (s *server) handleChatStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(s.chat.get())
}
