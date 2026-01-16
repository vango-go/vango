package server

import "strings"

// ValidateExternalRedirectURL validates an absolute redirect URL against an allowlist.
// Returns the canonical URL and true when allowed; otherwise returns ("", false).
func ValidateExternalRedirectURL(rawURL string, allowedHosts []string) (string, bool) {
	allowlist := normalizeRedirectAllowlist(allowedHosts)
	return validateExternalRedirect(rawURL, allowlist)
}

func normalizeRedirectAllowlist(allowedHosts []string) map[string]struct{} {
	if len(allowedHosts) == 0 {
		return nil
	}
	allowlist := make(map[string]struct{}, len(allowedHosts))
	for _, host := range allowedHosts {
		h := strings.ToLower(strings.TrimSpace(host))
		if h == "" {
			continue
		}
		allowlist[h] = struct{}{}
	}
	return allowlist
}
