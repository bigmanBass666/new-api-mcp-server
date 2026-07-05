package extractor

import "net/http"

// addAuthHeaders sets the Authorization and New-Api-User headers on a request
// using the extractor's configuration.
func (e *Extractor) addAuthHeaders(req *http.Request) {
	if e.cfg.SystemKey != "" {
		req.Header.Set("Authorization", "Bearer "+e.cfg.SystemKey)
	}
	if e.cfg.UserID != "" {
		req.Header.Set("New-Api-User", e.cfg.UserID)
	}
}