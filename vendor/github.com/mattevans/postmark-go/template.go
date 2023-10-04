package postmark

import (
	"fmt"
	"net/url"
	"strconv"
)

const (
	templatesAPIPath = "templates"
)

// TemplateService handles communication with the template related methods of the
// Postmark API (https://postmarkapp.com/developer/api/templates-api)
type TemplateService service

// Templates represents a slice of templates and a given count.
type Templates struct {
	TotalCount int64
	Templates  []TemplateOverview
}

// TemplateOverview represents overview/identifying information about a template.
type TemplateOverview struct {
	TemplateId int64
	Name       string
	Active     bool
}

// Template represents a templates in further detail.
type Template struct {
	TemplateId         int64
	Name               string
	Subject            string
	HtmlBody           string
	TextBody           string
	AssociatedServerId int64
	Active             bool
}

/*
 * GetTemplates --------------------------------------------------------- */

// GetTemplates will return all templates.
func (s *TemplateService) GetTemplates(count, offset int) (*Templates, *Response, error) {
	// Construct query parameters.
	values := &url.Values{}
	values.Add("count", strconv.Itoa(count))
	values.Add("offset", strconv.Itoa(offset))

	request, err := s.client.NewRequest("GET", fmt.Sprintf("%s?%s", templatesAPIPath, values.Encode()), nil)
	if err != nil {
		return nil, nil, err
	}

	templates := &Templates{}
	response, err := s.client.Do(request, templates)
	if err != nil {
		return nil, response, err
	}

	return templates, response, nil
}

/*
 * GetSingleTemplate --------------------------------------------------------- */

// GetSingleTemplate will return a single template by ID.
func (s *TemplateService) GetSingleTemplate(templateID int64) (*Template, *Response, error) {
	request, err := s.client.NewRequest("GET", fmt.Sprintf("%s/%v", templatesAPIPath, templateID), nil)
	if err != nil {
		return nil, nil, err
	}

	template := Template{}
	response, err := s.client.Do(request, &template)
	if err != nil {
		return nil, nil, err
	}

	return &template, response, nil
}
