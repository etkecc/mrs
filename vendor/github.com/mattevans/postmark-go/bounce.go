package postmark

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

const (
	bounceDeliveryStatsAPIPath = "deliverystats"
	bounceBouncesAPIPath       = "bounces"
)

// BounceService handles communication with the bounce related methods of the
// Postmark API (http://developer.postmarkapp.com/developer-api-bounce.html)
type BounceService service

// BounceType represents the type of bounce with a count.
type BounceType struct {
	Type  string
	Name  string
	Count int64
}

// BounceDeliveryStats represents a collection of bounce stats.
type BounceDeliveryStats struct {
	InactiveMails int64
	Bounces       []BounceType
}

// Bounces represents a slice of bounces and a given count.
type Bounces struct {
	TotalCount int64
	Bounces    []Bounce
}

// Bounce represents a BounceType in further detail.
type Bounce struct {
	ID            int64
	Type          string
	TypeCode      int64
	Name          string
	Tag           string
	MessageID     string
	Description   string
	Details       string
	Email         string
	BouncedAt     time.Time
	DumpAvailable bool
	Inactive      bool
	CanActivate   bool
	Subject       string
}

// BounceDump represents a the raw source of bounce.
type BounceDump struct {
	Body string
}

// BounceActivated represents a bounce that has been reactivated.
type BounceActivated struct {
	Message string
	Bounce  Bounce
}

/*
 * GetDeliveryStats --------------------------------------------------------- */

// GetDeliveryStats will return all delivery stats aggregated by bounce type.
func (s *BounceService) GetDeliveryStats() (*BounceDeliveryStats, *Response, error) {
	request, err := s.client.NewRequest("GET", bounceDeliveryStatsAPIPath, nil)
	if err != nil {
		return nil, nil, err
	}

	deliveryStats := &BounceDeliveryStats{}
	response, err := s.client.Do(request, deliveryStats)
	if err != nil {
		return nil, response, err
	}

	return deliveryStats, response, nil
}

/*
 * GetBounces --------------------------------------------------------- */

// GetBounces will return all bounces.
func (s *BounceService) GetBounces(bounceCount, bounceOffset int, parameters map[string]interface{}) (*Bounces, *Response, error) {

	// Ensure our bounce count meets criteria.
	if bounceCount > 500 {
		return nil, nil, errors.New("The max number of bounces to return per request is 500")
	}

	// Construct query parameters.
	values := &url.Values{}
	values.Add("count", strconv.Itoa(bounceCount))
	values.Add("offset", strconv.Itoa(bounceOffset))
	for key, value := range parameters {
		values.Add(key, fmt.Sprintf("%v", value))
	}

	request, err := s.client.NewRequest("GET", fmt.Sprintf("%s?%s", bounceBouncesAPIPath, values.Encode()), nil)
	if err != nil {
		return nil, nil, err
	}

	bounces := Bounces{}
	response, err := s.client.Do(request, &bounces)
	if err != nil {
		return nil, nil, err
	}

	return &bounces, response, nil
}

/*
 * GetSingleBounce --------------------------------------------------------- */

// GetSingleBounce will return a single bounce by ID.
func (s *BounceService) GetSingleBounce(bounceID int64) (*Bounce, *Response, error) {
	request, err := s.client.NewRequest("GET", fmt.Sprintf("%s/%v", bounceBouncesAPIPath, bounceID), nil)
	if err != nil {
		return nil, nil, err
	}

	bounce := Bounce{}
	response, err := s.client.Do(request, &bounce)
	if err != nil {
		return nil, nil, err
	}

	return &bounce, response, nil
}

/*
 * GetBounceDump --------------------------------------------------------- */

// GetBounceDump will return a single bounce dump by ID.
func (s *BounceService) GetBounceDump(bounceID int64) (*BounceDump, *Response, error) {
	request, err := s.client.NewRequest("GET", fmt.Sprintf("%s/%v/dump", bounceBouncesAPIPath, bounceID), nil)
	if err != nil {
		return nil, nil, err
	}

	dump := BounceDump{}
	response, err := s.client.Do(request, &dump)
	if err != nil {
		return nil, nil, err
	}

	return &dump, response, nil
}

/*
 * ActivateBounce --------------------------------------------------------- */

// ActivateBounce will attempt to reactivate this email via bounce ID.
func (s *BounceService) ActivateBounce(bounceID int64) (*BounceActivated, *Response, error) {
	request, err := s.client.NewRequest("PUT", fmt.Sprintf("%s/%v/activate", bounceBouncesAPIPath, bounceID), nil)
	if err != nil {
		return nil, nil, err
	}

	bounce := BounceActivated{}
	response, err := s.client.Do(request, &bounce)
	if err != nil {
		return nil, nil, err
	}

	return &bounce, response, nil
}

/*
 * GetBounceTags --------------------------------------------------------- */

// GetBounceTags will return a slice of tag values that have generated bounces.
func (s *BounceService) GetBounceTags() ([]string, *Response, error) {
	request, err := s.client.NewRequest("GET", fmt.Sprintf("%s/tags", bounceBouncesAPIPath), nil)
	if err != nil {
		return nil, nil, err
	}

	tags := []string{}
	response, err := s.client.Do(request, &tags)
	if err != nil {
		return nil, nil, err
	}

	return tags, response, nil
}
