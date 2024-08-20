package matrix

import (
	"net/url"
	"time"

	"gitlab.com/etke.cc/mrs/api/model"
	"gitlab.com/etke.cc/mrs/api/utils"
	"gitlab.com/etke.cc/mrs/api/version"
)

func (s *Server) initKeys(strs []string) error {
	if len(strs) == 0 {
		return nil
	}
	keys := []*model.Key{}
	for _, str := range strs {
		key, err := model.KeyFrom(str)
		if err != nil {
			return err
		}
		keys = append(keys, key)
	}
	s.keys = keys
	return nil
}

func (s *Server) initWellKnown(apiURL string) error {
	uri, err := url.Parse(apiURL)
	if err != nil {
		return err
	}
	port := uri.Port()
	if port == "" {
		port = "443"
	}

	serverValue, err := utils.JSON(map[string]string{
		"s.server": uri.Hostname() + ":" + port,
	})
	if err != nil {
		return err
	}
	s.wellknownServer = serverValue

	clientValue, err := utils.JSON(map[string]map[string]string{
		"s.homeserver": {
			"base_url": "https://" + uri.Host,
		},
	})
	if err != nil {
		return err
	}
	s.wellknownClient = clientValue

	supportValue, err := utils.JSON(s.cfg.Get().Matrix.Support)
	if err != nil {
		return err
	}
	s.wellknownSupport = supportValue
	return nil
}

func (s *Server) initVersion() error {
	serverValue, err := utils.JSON(serverVersionResp{
		Server: map[string]string{
			"name":    version.Name,
			"version": version.Version,
		},
	})
	if err != nil {
		return err
	}
	s.versionServer = serverValue
	clientValue, err := utils.JSON(clientVersionResp{
		Versions: []string{ // copy of synapse versions with some made-up values, because MRS is not a client-facing server, but other services like matrix.to require such hacks.
			"r0.0.1",
			"r0.1.0",
			"r0.2.0",
			"r0.3.0",
			"r0.4.0",
			"r0.5.0",
			"r0.6.0",
			"r0.6.1",
			"v1.1",
			"v1.2",
			"v1.3",
			"v1.4",
			"v1.5",
			"v1.6",
			"v1.7",
			"v1.8",
			"v1.9",
			"v1.10",
		},
		UnstableFeatures: map[string]bool{
			"uk.half-shot.msc1929":     true, // the name is made-up as well, because the MSC itself doesn't contain any name
			"support.feline.msc4121":   true,
			"is.nheko.summary.msc3266": true,
		},
	})
	if err != nil {
		return err
	}
	s.versionClient = clientValue
	return nil
}

func (s *Server) initKeyServer() {
	resp := matrixKeyResp{
		ServerName:    s.cfg.Get().Matrix.ServerName,
		ValidUntilTS:  time.Now().UTC().Add(24 * time.Hour).UnixMilli(),
		VerifyKeys:    map[string]map[string]string{},
		OldVerifyKeys: map[string]map[string]any{},
	}
	for _, key := range s.keys {
		resp.VerifyKeys[key.ID] = map[string]string{"key": key.Public}
	}
	s.keyServer = resp
}
