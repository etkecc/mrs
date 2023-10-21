package robots

import "strings"

type parser struct {
	agents      []*agent
	withinGroup bool
	items       []*item
	robotsdata  *robotsdata
}

type parsefn func(p *parser) parsefn

func parse(s string) *robotsdata {
	p := &parser{
		items:      lex(s),
		robotsdata: &robotsdata{},
	}
	for fn := parseStart; fn != nil; fn = fn(p) {
	}
	return p.robotsdata
}

func parseStart(p *parser) parsefn {
	switch p.items[0].typ {
	case itemUserAgent:
		return parseUserAgent
	case itemDisallow:
		return parseDisallow
	case itemAllow:
		return parseAllow
	case itemSitemap:
		return parseSitemap
	default:
		return parseNext
	}
}

// parseUserAgent handles two important cases. First, if we are within
// a group of rules already, a user-agent rule causes a new group to
// begin.  Second, if we're starting a new group (i.e., the previous
// rule was also a user-agent rule and we're associating another agent
// with the forthcoming group) then we add another agent to p.agents.
func parseUserAgent(p *parser) parsefn {
	if p.withinGroup { // The previous rule was allow or disallow
		p.robotsdata.addAgents(p.agents)
		p.agents = []*agent{
			&agent{
				name: p.items[0].val,
			},
		}
		p.withinGroup = false // Now we're before the start of a group
		return parseNext
	}
	// The previous rule was another user-agent rule
	p.agents = append(p.agents, &agent{
		name: p.items[0].val,
	})
	return parseNext
}

// parseAllow and parseDisallow are identical except for what they set
// the allow field of the member to. Therefore, we have this factory
// function.
func makeParseMember(allow bool) func(*parser) parsefn {
	return func(p *parser) parsefn {
		// Note that we set withinGroup to true even if we're
		// evaluating allow/disallow rules that come before
		// any user-agent rules. That's fine, it results in
		// the desired behavior.
		p.withinGroup = true
		// If there is no path, do nothing.
		if strings.TrimSpace(p.items[0].val) == "" {
			return parseNext
		}
		// If there is no agent (i.e., the rules come before
		// any user-agent line), this just doesn't do
		// anything.  That's what we want.
		for _, agent := range p.agents {
			m := &member{
				allow: allow,
				path:  p.items[0].val,
			}
			agent.group.addMember(m)
		}
		return parseNext
	}
}

var parseDisallow parsefn

var parseAllow parsefn

func init() {
	// These variables must be initiated at run-time to avoid a
	// definition loop.
	parseDisallow = makeParseMember(false)
	parseAllow = makeParseMember(true)
}

func parseSitemap(p *parser) parsefn {
	// sitemap rules are global: they do not affect whether we are
	// in a group or not.
	p.robotsdata.sitemaps = append(p.robotsdata.sitemaps, p.items[0].val)
	return parseNext
}

func parseNext(p *parser) parsefn {
	p.items = p.items[1:]
	if len(p.items) == 0 {
		return parseEnd
	}
	return parseStart
}

func parseEnd(p *parser) parsefn {
	p.robotsdata.addAgents(p.agents)
	return nil
}
