package services

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"sync"

	"github.com/benjaminestes/robots/v2"

	"github.com/etkecc/mrs/internal/utils"
	"github.com/etkecc/mrs/internal/version"
)

const (
	// RobotsTxtPublicRooms is matrix federation public rooms endpoint
	RobotsTxtPublicRooms = "/_matrix/federation/v1/publicRooms"
	// RobotsTxtPublicRoom is made up endpoint of a specific matrix room, as there is no better option
	RobotsTxtPublicRoom = "/_matrix/federation/v1/publicRooms/%s"
)

var robotsTxtBot = []byte(version.Bot)

// Robots - robots.txt parsing
// Deprecated: This service is deprecated and will be removed in future versions
type Robots struct {
	mu   *sync.Mutex
	data map[string]*robots.Robots
}

// NewRobots creates robots.txt parsing service
func NewRobots() *Robots {
	return &Robots{
		mu:   &sync.Mutex{},
		data: make(map[string]*robots.Robots),
	}
}

// Allowed checks if endpoint is allowed by robots.txt of the serverName
// Deprecated: This service is deprecated and will be removed in future versions
func (r *Robots) Allowed(ctx context.Context, serverName, endpoint string) bool {
	parsed := r.get(ctx, serverName)
	if parsed == nil {
		return true
	}

	return parsed.Test(version.Bot, endpoint)
}

// isEligible checks if robots.txt response is eligible for parsing
func (r *Robots) isEligible(resp *http.Response) bool {
	if resp.StatusCode != http.StatusOK {
		return false
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return false
	}
	if !bytes.Contains(body, robotsTxtBot) {
		return false
	}

	resp.Body = io.NopCloser(bytes.NewBuffer(body))
	return true
}

// parse robots.txt by server name
func (r *Robots) parse(ctx context.Context, serverName string) {
	defer func() {
		if err := recover(); err != nil {
			r.set(serverName, nil)
		}
	}()

	robotsURL, err := robots.Locate("https://" + serverName + "/")
	if err != nil {
		r.set(serverName, nil)
		return
	}

	resp, err := utils.Get(ctx, robotsURL)
	if err != nil {
		r.set(serverName, nil)
		return
	}

	if !r.isEligible(resp) {
		r.set(serverName, nil)
		return
	}
	defer resp.Body.Close()

	parsed, err := robots.From(resp.StatusCode, resp.Body)
	if err != nil {
		r.set(serverName, nil)
		return
	}
	r.set(serverName, parsed)
}

// set parsed robots.txt
func (r *Robots) set(serverName string, parsed *robots.Robots) {
	r.mu.Lock()
	r.data[serverName] = parsed
	r.mu.Unlock()
}

// get parsed robots.txt
func (r *Robots) get(ctx context.Context, serverName string) *robots.Robots {
	r.mu.Lock()
	_, ok := r.data[serverName]
	r.mu.Unlock()

	if !ok {
		r.parse(ctx, serverName)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	return r.data[serverName]
}
