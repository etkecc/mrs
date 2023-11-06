package services

import (
	"fmt"
	"regexp"

	"gitlab.com/etke.cc/mrs/api/model"
	"gitlab.com/etke.cc/mrs/api/utils"
)

// based on W3C email regex, ref: https://www.w3.org/TR/2016/REC-html51-20161101/sec-forms.html#email-state-typeemail
var domainRegex = regexp.MustCompile(`^(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z0-9][a-zA-Z0-9-]{0,61}[a-zA-Z0-9]$`)

// Validator is matrix validation service
type Validator struct {
	cfg    ConfigService
	block  BlocklistService
	matrix FederationService
	robots RobotsService
}

// NewValidator creates new validation service
func NewValidator(cfg ConfigService, block BlocklistService, matrix FederationService, robots RobotsService) *Validator {
	return &Validator{
		cfg:    cfg,
		block:  block,
		matrix: matrix,
		robots: robots,
	}
}

// Domain checks if domain name is valid
func (v *Validator) Domain(server string) bool {
	// own server
	if v.cfg.Get().Matrix.ServerName == server {
		return false
	}

	// check if domain is valid
	if len(server) < 4 || len(server) > 77 {
		return false
	}

	// check if domain is valid
	if !domainRegex.MatchString(server) {
		return false
	}

	return true
}

// IsOnline checks if matrix server is online and federatable
func (v *Validator) IsOnline(server string) (string, bool) {
	// check if domain is valid
	if !v.Domain(server) {
		return "", false
	}

	// check if online
	name, err := v.matrix.QueryServerName(server)
	if name == "" || err != nil {
		return "", false
	}

	// check if not blocked
	if v.block.ByServer(name) {
		return "", false
	}

	// check if federateable
	if _, _, err := v.matrix.QueryVersion(server); err != nil {
		return "", false
	}

	return name, true
}

// IsIndexable check if server is indexable
func (v *Validator) IsIndexable(server string) bool {
	log := utils.Logger.With().Str("server", server).Logger()
	if !v.Domain(server) {
		log.Info().Str("reason", "domain").Msg("not indexable")
		return false
	}
	if !v.robots.Allowed(server, RobotsTxtPublicRooms) {
		log.Info().Str("reason", "robots.txt").Msg("not indexable")
		return false
	}
	if _, err := v.matrix.QueryPublicRooms(server, "1", ""); err != nil {
		log.Info().Err(err).Str("reason", "publicRooms").Msg("not indexable")
		return false
	}
	log.Info().Msg("indexable")
	return true
}

// IsRoomAllowed checks if room is allowed
func (v *Validator) IsRoomAllowed(server string, room *model.MatrixRoom) bool {
	if room.ID == "" {
		return false
	}
	if v.block.ByID(room.ID) {
		return false
	}
	if v.block.ByID(room.Alias) {
		return false
	}
	if v.block.ByServer(room.Server) {
		return false
	}
	if v.block.ByServer(server) {
		return false
	}

	return v.robots.Allowed(server, fmt.Sprintf(RobotsTxtPublicRoom, room.ID))
}
