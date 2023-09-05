package proxy

import (
	"fmt"

	"golang.org/x/exp/maps"
)

type ProxyRegistry struct {
	data map[ProxyScheme]ProxyDescription
}

func (r ProxyRegistry) Keys() []ProxyScheme {
	return maps.Keys(r.data)
}

func (r ProxyRegistry) Values() []ProxyDescription {
	return maps.Values(r.data)
}

func (r ProxyRegistry) Get(key ProxyScheme) (ProxyDescription, error) {
	if v, ok := r.data[key]; ok {
		return v, nil
	}
	return ProxyDescription{}, fmt.Errorf("no such proxy: %s", key)
}

func (r *ProxyRegistry) Add(desc ProxyDescription) {
	if _, ok := r.data[desc.Scheme]; ok {
		panic(fmt.Sprintf("proxy with scheme %s already registered", desc.Scheme))
	}

	r.data[desc.Scheme] = desc
}

func (r *ProxyRegistry) FindAndCreateProxy(addr *ProxyAddr) (*ProxyDescription, error) {
	p, err := r.Get(addr.ProxyScheme())
	if err != nil {
		return nil, err
	}

	return p.SetAddr(addr), nil
}

var Registry = ProxyRegistry{data: make(map[ProxyScheme]ProxyDescription)}
