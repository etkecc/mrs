package services

import (
	"context"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/benjaminestes/robots/v2"

	"gitlab.com/etke.cc/mrs/api/version"
)

const (
	// RobotsTxtPublicRooms is matrix federation public rooms endpoint
	RobotsTxtPublicRooms = "/_matrix/federation/v1/publicRooms"
	// RobotsTxtPublicRoom is made up endpoint of a specific matrix room, as there is no better option
	RobotsTxtPublicRoom = "/_matrix/federation/v1/publicRooms/%s"
)

// Robots - robots.txt parsing
type Robots struct {
	mu     *sync.Mutex
	data   map[string]*robots.Robots
	client *http.Client
}

// NewRobots creates robots.txt parsing service
func NewRobots() *Robots {
	dialer := &net.Dialer{
		Timeout: 5 * time.Second,
	}

	return &Robots{
		mu:   &sync.Mutex{},
		data: make(map[string]*robots.Robots),
		client: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        1,
				MaxConnsPerHost:     1,
				MaxIdleConnsPerHost: 1,
				TLSHandshakeTimeout: 10 * time.Second,
				DialContext:         dialer.DialContext,
				Dial:                dialer.Dial,
			},
		},
	}
}

// Allowed checks if endpoint is allowed by robots.txt of the serverName
func (r *Robots) Allowed(serverName, endpoint string) bool {
	parsed := r.get(serverName)
	if parsed == nil {
		return true
	}

	return parsed.Test(version.Bot, endpoint)
}

// parse robots.txt by server name
func (r *Robots) parse(serverName string) {
	defer func() {
		if err := recover(); err != nil {
			log.Println(serverName, "robots.txt parser paniced", err)
		}
	}()

	robotsURL, err := robots.Locate("https://" + serverName + "/")
	if err != nil {
		log.Println(serverName, "cannot locate robots.txt", err)
		r.set(serverName, nil)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, robotsURL, nil)
	if err != nil {
		log.Println(serverName, "cannot create robots.txt request", err)
		r.set(serverName, nil)
		return
	}
	req.Header.Set("User-Agent", version.UserAgent)
	resp, err := r.client.Do(req)
	if err != nil {
		log.Println(serverName, "cannot get robots.txt", err)
		r.set(serverName, nil)
		return
	}
	defer resp.Body.Close()

	parsed, err := robots.From(resp.StatusCode, resp.Body)
	if err != nil {
		log.Println(serverName, "cannot read robots.txt", err)
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
func (r *Robots) get(serverName string) *robots.Robots {
	r.mu.Lock()
	_, ok := r.data[serverName]
	r.mu.Unlock()

	if !ok {
		r.parse(serverName)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	return r.data[serverName]
}
