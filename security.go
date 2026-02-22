package main

import (
	"net"
	"net/http"
	"net/url"
	"strings"
)

var allowedWebSocketOrigins []string

func isAllowedWebSocketOrigin(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		// Non-browser clients may omit Origin.
		return true
	}

	if originAllowedByConfig(origin) {
		return true
	}

	u, err := url.Parse(origin)
	if err != nil || u.Hostname() == "" {
		return false
	}

	return strings.EqualFold(u.Hostname(), hostOnly(r.Host))
}

func setAllowedWebSocketOriginsFromCSV(raw string) {
	allowedWebSocketOrigins = allowedWebSocketOrigins[:0]
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		allowedWebSocketOrigins = append(allowedWebSocketOrigins, item)
	}
}

func originAllowedByConfig(origin string) bool {
	for _, item := range allowedWebSocketOrigins {
		if strings.EqualFold(item, origin) {
			return true
		}
	}
	return false
}

func hostOnly(hostport string) string {
	hostport = strings.TrimSpace(hostport)
	if hostport == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(hostport); err == nil {
		return strings.Trim(strings.ToLower(host), "[]")
	}
	return strings.Trim(strings.ToLower(hostport), "[]")
}
