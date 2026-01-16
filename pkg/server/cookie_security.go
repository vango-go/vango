package server

import (
	"log/slog"
	"net"
	"net/http"
	"strings"
)

// CookiePolicy enforces cookie defaults and security requirements.
type CookiePolicy struct {
	config         *ServerConfig
	trustedProxies *proxyMatcher
	logger         *slog.Logger
}

// CookiePolicyOptions configures optional overrides for cookie policy.
type CookiePolicyOptions struct {
	httpOnly *bool
}

// CookieOption configures cookie policy application.
type CookieOption func(*CookiePolicyOptions)

// WithCookieHTTPOnly overrides the default HttpOnly behavior for a cookie.
func WithCookieHTTPOnly(enabled bool) CookieOption {
	return func(opts *CookiePolicyOptions) {
		opts.httpOnly = &enabled
	}
}

func newCookiePolicy(config *ServerConfig, trustedProxies *proxyMatcher, logger *slog.Logger) *CookiePolicy {
	if logger == nil {
		logger = slog.Default()
	}
	return &CookiePolicy{
		config:         config,
		trustedProxies: trustedProxies,
		logger:         logger,
	}
}

// Apply applies cookie defaults and enforces SecureCookies requirements.
// Returns ErrSecureCookiesRequired if secure cookies are enabled but the request is not secure.
func (p *CookiePolicy) Apply(r *http.Request, cookie *http.Cookie, opts ...CookieOption) (*http.Cookie, error) {
	if p == nil || cookie == nil {
		return cookie, nil
	}
	if p.config == nil {
		return cookie, nil
	}

	var options CookiePolicyOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}

	if p.config.SecureCookies {
		if !p.isRequestSecure(r) {
			return nil, ErrSecureCookiesRequired
		}
		cookie.Secure = true
	}

	if cookie.SameSite == 0 || cookie.SameSite == http.SameSiteDefaultMode {
		cookie.SameSite = p.config.SameSiteMode
	}
	if cookie.Domain == "" && p.config.CookieDomain != "" {
		cookie.Domain = p.config.CookieDomain
	}
	if options.httpOnly != nil {
		cookie.HttpOnly = *options.httpOnly
	} else if p.config.CookieHTTPOnly && !cookie.HttpOnly {
		cookie.HttpOnly = true
	}

	return cookie, nil
}

// ApplyCookiePolicy applies defaults without overrides. This is a compatibility hook for
// external packages that accept a generic cookie policy interface.
func (p *CookiePolicy) ApplyCookiePolicy(r *http.Request, cookie *http.Cookie) (*http.Cookie, error) {
	return p.Apply(r, cookie)
}

func (p *CookiePolicy) isRequestSecure(r *http.Request) bool {
	if r == nil {
		return false
	}
	if r.TLS != nil {
		return true
	}
	if !p.isTrustedProxy(r) {
		return false
	}

	if proto := forwardedProto(r.Header.Get("Forwarded")); proto != "" {
		return isSecureProto(proto)
	}
	if proto := forwardedProtoValue(r.Header.Get("X-Forwarded-Proto")); proto != "" {
		return isSecureProto(proto)
	}

	return false
}

func (p *CookiePolicy) isTrustedProxy(r *http.Request) bool {
	if p.trustedProxies == nil {
		return false
	}
	return p.trustedProxies.IsTrusted(remoteIPFromRequest(r))
}

func (s *Server) csrfCookie(r *http.Request, token string) (*http.Cookie, error) {
	cookie := &http.Cookie{
		Name:     CSRFCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: false, // Must be readable by JS for Double Submit
	}
	if s.cookiePolicy == nil {
		return cookie, nil
	}
	return s.cookiePolicy.Apply(r, cookie, WithCookieHTTPOnly(false))
}

func forwardedProto(header string) string {
	if header == "" {
		return ""
	}
	parts := strings.Split(header, ",")
	first := strings.TrimSpace(parts[0])
	if first == "" {
		return ""
	}
	for _, param := range strings.Split(first, ";") {
		param = strings.TrimSpace(param)
		if param == "" {
			continue
		}
		kv := strings.SplitN(param, "=", 2)
		if len(kv) != 2 {
			continue
		}
		if strings.EqualFold(kv[0], "proto") {
			value := strings.TrimSpace(kv[1])
			value = strings.Trim(value, "\"")
			return strings.ToLower(value)
		}
	}
	return ""
}

func forwardedProtoValue(header string) string {
	if header == "" {
		return ""
	}
	parts := strings.Split(header, ",")
	value := strings.TrimSpace(parts[0])
	value = strings.Trim(value, "\"")
	return strings.ToLower(value)
}

func isSecureProto(proto string) bool {
	switch strings.ToLower(proto) {
	case "https", "wss":
		return true
	default:
		return false
	}
}

func remoteIPFromRequest(r *http.Request) net.IP {
	if r == nil {
		return nil
	}
	host := strings.TrimSpace(r.RemoteAddr)
	if host == "" {
		return nil
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	if zone := strings.Index(host, "%"); zone != -1 {
		host = host[:zone]
	}
	return net.ParseIP(host)
}

type proxyMatcher struct {
	ips  map[string]struct{}
	nets []*net.IPNet
}

func newProxyMatcher(entries []string, logger *slog.Logger) *proxyMatcher {
	if len(entries) == 0 {
		return nil
	}

	ips := make(map[string]struct{})
	var nets []*net.IPNet

	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if strings.Contains(entry, "/") {
			_, network, err := net.ParseCIDR(entry)
			if err != nil {
				if logger != nil {
					logger.Warn("invalid trusted proxy CIDR", "entry", entry, "error", err)
				}
				continue
			}
			nets = append(nets, network)
			continue
		}
		ip := net.ParseIP(entry)
		if ip == nil {
			if logger != nil {
				logger.Warn("invalid trusted proxy IP", "entry", entry)
			}
			continue
		}
		ips[ip.String()] = struct{}{}
	}

	if len(ips) == 0 && len(nets) == 0 {
		return nil
	}

	return &proxyMatcher{ips: ips, nets: nets}
}

func (m *proxyMatcher) IsTrusted(ip net.IP) bool {
	if m == nil || ip == nil {
		return false
	}
	if len(m.ips) > 0 {
		if _, ok := m.ips[ip.String()]; ok {
			return true
		}
	}
	for _, network := range m.nets {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}
