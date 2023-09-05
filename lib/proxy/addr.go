package proxy

import (
	"net/url"
	"strconv"
)

type ProxyAddr struct {
	url.URL
}

func ParseAddr(raw string) (*ProxyAddr, error) {
	u, err := url.Parse(fixupURL(raw))
	if err != nil {
		return nil, err
	}
	return &ProxyAddr{*u}, nil
}

func (a *ProxyAddr) ProxyScheme() ProxyScheme {
	return ProxyScheme(a.Scheme)
}

func (a *ProxyAddr) Network() string {
	return string(a.ProxyScheme())
}

func (a *ProxyAddr) GetStringOption(key, fallback string) string {
	// URL elements not in querystring.
	switch key {
	case "Hostname":
		if a.Hostname() == "" {
			return fallback
		}
		return a.Hostname()
	case "Port":
		if a.Port() == "" {
			return fallback
		}
		return a.Port()
	case "Path":
		if a.Path == "" {
			return fallback
		}
		return a.Path
	}

	qs := a.URL.Query()

	if qs.Has(key) {
		return qs.Get(key)
	}
	return fallback
}

func (a *ProxyAddr) GetBoolOption(key string, fallback bool) (bool, error) {
	qs := a.URL.Query()

	if qs.Has(key) {
		return strconv.ParseBool(qs.Get(key))
	}
	return fallback, nil
}

func (a *ProxyAddr) GetIntOption(key string, base int, fallback int) (int, error) {
	qs := a.URL.Query()

	if qs.Has(key) {
		v, err := strconv.ParseInt(qs.Get(key), 32, base)
		return int(v), err
	}
	return fallback, nil
}

func (a *ProxyAddr) String() string {
	return a.URL.String()
}
