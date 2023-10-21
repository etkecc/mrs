package robots

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// member represents a group-member record as defined in Google's
// specification. After its path has been set, the compile() method
// must be called prior to use.
type member struct {
	allow   bool
	path    string
	pattern *regexp.Regexp
}

// Check whether the given path is matched by this record.
func (m *member) match(path string) bool {
	return m.pattern.MatchString(path)
}

// A group-member record specifies a path to which it
// applies. Internally to this package, we need an efficient way of
// matching that path, which possibly includes metacharacters * and
// $. compile() compiles the given path to a regular expression
// denoting the patterns we will accept.
func (m *member) compile() {
	// This approach to handling matches is derived from temoto's:
	// https://github.com/temoto/robotstxt/blob/master/parser.go
	pattern := regexp.QuoteMeta(m.path)
	pattern = "^" + pattern // But with an added start-of-line
	pattern = strings.Replace(pattern, `\*`, `.*`, -1)
	pattern = strings.Replace(pattern, `\$`, `$`, -1)
	// FIXME: What do I do in case of error?
	r, err := regexp.Compile(pattern)
	if err != nil {
		fmt.Printf("oh no! %v\n", err)
	}
	m.pattern = r
}

// A group is an ordered list of members. The members are ordered from
// longest path to shortest path. This allows efficient matching of
// paths to members: when evaluated sequentially, the first match must
// be the longest.
type group struct {
	members []*member
}

func (g *group) addMember(m *member) {
	// Maintain type invariant: a member must have its pattern
	// compiled before use.
	m.compile()
	// Maintain type invariant: the members of a group must always
	// be sorted by length of path, descending.
	g.members = insertMemberMaintainingOrder(g.members, m)
}

func insertMemberMaintainingOrder(a []*member, m *member) []*member {
	a = append(a, m)
	for i := len(a) - 1; i > 0; i-- {
		if len(a[i].path) < len(a[i-1].path) {
			return a
		}
		a[i], a[i-1] = a[i-1], a[i]
	}
	return a
}

// An agent represents a group of rules that a named robots agent
// might match. Its compile() method must be called prior to use.
type agent struct {
	name    string
	group   group
	pattern *regexp.Regexp
}

// Test whether the given robots agent string matches this agent.
func (a *agent) match(name string) bool {
	return a.pattern.MatchString(strings.ToLower(name))
}

// A agent specifies a robots agent to which it applies. Internally to
// this package, we need an efficient way of matching that name, which
// includes no metacharacters. However, we will treat the special case
// "*" as matching all agents for which no other match
// exists. compile() compiles the amended name to a regular expression
// denoting the patterns we will accept.
func (a *agent) compile() {
	pattern := regexp.QuoteMeta(a.name)
	if pattern == `\*` {
		pattern = strings.Replace(pattern, `\*`, `.*`, -1)
	}
	pattern = "^" + pattern
	pattern = strings.ToLower(pattern)
	// FIXME: What do I do in case of error?
	r, err := regexp.Compile(pattern)
	if err != nil {
		fmt.Printf("oh no! %v\n", err)
	}
	a.pattern = r
}

// robotsdata represents the result of parsing a robots.txt file. To
// test whether an agent may crawl a path, use a Test* method. If any
// sitemaps were discovered while parsing, the Sitemaps field will be
// a slice containing their absolute URLs.
type robotsdata struct {
	// agents represents the groups of rules from a robots
	// file. The agents occur in descending order by length of
	// name. This ensures that if we check the agents
	// sequentially, the first matching agent will be the longest
	// match as well.
	agents   []*agent
	sitemaps []string // Absolute URLs of sitemaps in robots.txt.
}

// Robots represents an object whose methods govern access to URLs
// within the scope of a robots.txt file, and what sitemaps, if any,
// have been discovered during parsing.
type Robots struct {
	allow bool // default crawl setting
	*robotsdata
}

func makeRobots(status int, data *robotsdata) *Robots {
	if data == nil {
		// If a nil data pointer is passed, just construct an
		// empty data object. This avoids having to check for
		// nil pointers while allowing for the possibility
		// that there was actually no robots.txt data.
		data = &robotsdata{}
	}
	r := &Robots{
		robotsdata: data,
	}
	r.setAllow(status)
	return r
}

func (r *Robots) setAllow(status int) {
	if status >= 500 && status < 600 {
		r.allow = false
		return
	}
	r.allow = true
	return
}

// bestAgent matches an agent string against all of the agents in
// r. It returns a pointer to the best matching agent, and a boolen
// indicating whether a match was found.
func (r *Robots) bestAgent(name string) (*agent, bool) {
	for _, agent := range r.agents {
		if agent.match(name) {
			return agent, true
		}
	}
	return nil, false
}

// addAgents adds a slice of agents to that maintained by r.
// This function accepts a slice because that is the common case:
// the parser may generate multiple agent objects from a single
// group of rules.
func (r *robotsdata) addAgents(agents []*agent) {
	for _, agent := range agents {
		// Maintain type invariant: all contained agents
		// must have patterns compiled before use.
		agent.compile()
		// Maintain type invariant: r.agents must always be
		// sorted by length of agent name, descending.
		r.agents = insertAgentMaintainingOrder(r.agents, agent)
	}
}

func insertAgentMaintainingOrder(a []*agent, t *agent) []*agent {
	a = append(a, t)
	for i := len(a) - 1; i > 0; i-- {
		if len(a[i].name) < len(a[i-1].name) {
			return a
		}
		a[i], a[i-1] = a[i-1], a[i]
	}
	return a
}

// Sitemaps returns a list of sitemap URLs dicovered during parsing.
// The specification requires sitemap URLs in robots.txt files to be
// absolute, but this is the responsibility of the robots.txt author.
func (r *Robots) Sitemaps() []string {
	// TODO: Does this need to be immutable?
	return r.sitemaps
}

// Test takes an agent string and a rawurl string and checks whether the
// r allows name to access the path component of rawurl.
//
// Only the path of rawurl is used. For details, see method Tester.
func (r *Robots) Test(name, rawurl string) bool {
	return r.Tester(name)(rawurl)
}

// Tester takes string naming a user agent. It returns a predicate
// with a single string parameter representing a URL. This predicate
// can be used to check whether r allows name to crawl the path
// component of rawurl.
//
// Only the path part of rawurl is considered. Therefore, rawurl can
// be absolute or relative. It is the caller's responsibility to
// ensure that the Robots object is applicable to rawurl: no error can
// be provided if this is not the case. To ensure the Robots object is
// applicable to rawurl, use the Locate function.
func (r *Robots) Tester(name string) func(rawurl string) bool {
	agent, ok := r.bestAgent(name)
	if !ok {
		// An agent that isn't matched uses default allow state.
		return func(_ string) bool {
			return r.allow
		}
	}
	return func(rawurl string) bool {
		path, ok := robotsPath(rawurl)
		if !ok {
			return r.allow
		}
		for _, member := range agent.group.members {
			if member.match(path) {
				return member.allow
			}
		}
		// No applicable rule: return default robots allow state.
		return r.allow
	}
}

// robotsPath returns the part of a URL that robots.txt will match
// against.  This is the path, but also possibly a query string. The
// Path field of a parsed URL won't contain the query, so we
// concatenate it if it exists. It does not include a fragment.
func robotsPath(rawurl string) (string, bool) {
	parsed, err := url.Parse(rawurl)
	if err != nil {
		return "", false
	}
	path := parsed.Path
	if path == "" {
		path = "/"
	}
	if parsed.RawQuery != "" {
		path += "?" + parsed.RawQuery
	}
	return path, true
}
