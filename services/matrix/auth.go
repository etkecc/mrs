package matrix

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/goccy/go-json"
	"github.com/matrix-org/gomatrixserverlib"
	"github.com/rs/zerolog"

	"gitlab.com/etke.cc/mrs/api/utils"
)

// ValidateAuth validates matrix auth
func (s *Server) ValidateAuth(ctx context.Context, r *http.Request) (serverName string, err error) {
	span := utils.StartSpan(ctx, "matrix.ValidateAuth")
	defer span.Finish()
	log := zerolog.Ctx(span.Context())

	defer r.Body.Close()
	if s.cfg.Get().Matrix.ServerName == devhost {
		log.Warn().Msg("ignoring auth validation on dev host")
		return "ignored", nil
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return "", err
	}
	content := make(map[string]any)
	if len(body) > 0 {
		if jerr := json.Unmarshal(body, &content); jerr != nil {
			return "", jerr
		}
	}

	auths := s.parseAuths(span.Context(), r)
	if len(auths) == 0 {
		return "", fmt.Errorf("no auth provided")
	}
	obj := map[string]any{
		"method":      r.Method,
		"uri":         r.RequestURI,
		"origin":      auths[0].Origin,
		"destination": s.cfg.Get().Matrix.ServerName,
	}
	if len(content) > 0 {
		obj["content"] = content
	}
	canonical, err := utils.JSON(obj)
	if err != nil {
		return "", err
	}
	keys := s.queryKeys(span.Context(), auths[0].Origin)
	if len(keys) == 0 {
		return "", fmt.Errorf("no server keys available")
	}
	for _, auth := range auths {
		if err := s.validateAuth(obj, canonical, auth, keys); err != nil {
			log.
				Warn().
				Err(err).
				Str("canonical", string(canonical)).
				Any("content", content).
				Any("obj", obj).
				Msg("matrix auth validation failed")
			return "", err
		}
	}

	return auths[0].Origin, nil
}

// Authorize request
func (s *Server) Authorize(serverName, method, uri string, body any) ([]string, error) {
	obj := map[string]any{
		"method":      method,
		"uri":         uri,
		"origin":      s.cfg.Get().Matrix.ServerName,
		"destination": serverName,
	}
	if body != nil {
		obj["content"] = body
	}
	signed, err := s.signJSON(obj)
	if err != nil {
		return nil, err
	}
	var objSigned map[string]any
	if jerr := json.Unmarshal(signed, &objSigned); jerr != nil {
		return nil, jerr
	}
	if _, ok := objSigned["signatures"]; !ok {
		return nil, fmt.Errorf("no signatures")
	}
	allSignatures, ok := objSigned["signatures"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("cannot parse signatures: %v", objSigned["signatures"])
	}

	signatures, ok := allSignatures[s.cfg.Get().Matrix.ServerName].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("cannot parse own signatures: %v", allSignatures[s.cfg.Get().Matrix.ServerName])
	}
	headers := make([]string, 0, len(signatures))
	for keyID, sig := range signatures {
		headers = append(headers, fmt.Sprintf(
			`X-Matrix origin="%s",destination="%s",key="%s",sig="%s"`,
			s.cfg.Get().Matrix.ServerName, serverName, keyID, sig,
		))
	}
	return headers, nil
}

func (s *Server) parseAuth(ctx context.Context, authorization string) *matrixAuth {
	log := zerolog.Ctx(ctx)
	auth := &matrixAuth{}
	paramsStr := strings.ReplaceAll(authorization, "X-Matrix ", "")
	paramsSlice := strings.Split(paramsStr, ",")
	for _, param := range paramsSlice {
		parts := strings.Split(param, "=")
		if len(parts) < 2 {
			continue
		}
		value := strings.Trim(parts[1], `"`)
		switch parts[0] {
		case "origin":
			auth.Origin = value
		case "destination":
			destination := value
			auth.Destination = &destination
		case "key":
			auth.KeyID = value
		case "sig":
			sig, err := base64.RawStdEncoding.DecodeString(value)
			if err != nil {
				log.Warn().Err(err).Msg("cannot decode signature")
				return nil
			}
			auth.Signature = sig
		}
	}
	if auth.Origin == "" || auth.KeyID == "" || len(auth.Signature) == 0 {
		return nil
	}
	return auth
}

func (s *Server) validateAuth(obj map[string]any, canonical []byte, auth *matrixAuth, keys map[string]ed25519.PublicKey) error {
	if auth.Origin != obj["origin"] {
		return fmt.Errorf("auth is from multiple servers")
	}
	if auth.Destination != nil {
		if *auth.Destination != obj["destination"] {
			return fmt.Errorf("auth is for multiple servers")
		}

		if *auth.Destination != s.cfg.Get().Matrix.ServerName {
			return fmt.Errorf("unknown destination")
		}
	}

	key, ok := keys[auth.KeyID]
	if !ok {
		return fmt.Errorf("unknown key '%s'", auth.KeyID)
	}
	if !ed25519.Verify(key, canonical, auth.Signature) {
		return fmt.Errorf("failed signatures on '%s'", auth.KeyID)
	}

	return nil
}

// parseAuths parses Authorization headers,
// copied from https://github.com/turt2live/matrix-media-repo/blob/4da32e5739a8924e0cfcdde2daf4af4a90c2ff85/util/http.go#L52
func (s *Server) parseAuths(ctx context.Context, r *http.Request) []*matrixAuth {
	headers := r.Header.Values("Authorization")
	auths := make([]*matrixAuth, 0)
	for _, h := range headers {
		if !strings.HasPrefix(h, "X-Matrix ") {
			continue
		}
		auth := s.parseAuth(ctx, h)
		if auth != nil {
			auths = append(auths, auth)
		}
	}

	return auths
}

// signJSON using server keys
func (s *Server) signJSON(input any) ([]byte, error) {
	payload, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	for _, key := range s.keys {
		payload, err = gomatrixserverlib.SignJSON(s.cfg.Get().Matrix.ServerName, gomatrixserverlib.KeyID(key.ID), key.Private, payload)
		if err != nil {
			return nil, err
		}
	}
	return payload, nil
}
