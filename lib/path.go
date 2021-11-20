package jd

import (
	"fmt"
	"strconv"
)

type path []pathElement

type pathElement struct {
	key      pathKey
	metadata []Metadata
}

type pathKey interface {
	String() string
	isPathKey()
}

type metadataOnlyPathKey struct{}
type stringPathKey string
type indexPathKey int
type setElementPathKey struct{}
type specificSetElementPathKey struct{ obj jsonObject }

func (m metadataOnlyPathKey) String() string {
	return ""
}
func (s stringPathKey) String() string {
	return string(s)
}
func (i indexPathKey) String() string {
	return strconv.Itoa(int(i))
}
func (s setElementPathKey) String() string {
	return "{}"
}
func (s specificSetElementPathKey) String() string {
	return s.obj.Json()
}

func (metadataOnlyPathKey) isPathKey()       {}
func (stringPathKey) isPathKey()             {}
func (indexPathKey) isPathKey()              {}
func (setElementPathKey) isPathKey()         {}
func (specificSetElementPathKey) isPathKey() {}

func (p path) append(k pathKey, m ...Metadata) path {
	return p.appendElement(pathElement{k, m})
}

func (p path) appendMetadata(m ...Metadata) path {
	return p.appendElement(pathElement{metadataOnlyPathKey{}, m})
}

func (p path) appendElement(pe pathElement) (ret path) {
	if len(p) == 0 {
		return append(p, pe)
	}
	if _, ok := pe.key.(metadataOnlyPathKey); ok {
		return append(p, pe)
	}
	last := p[len(p)-1]
	if _, ok := last.key.(metadataOnlyPathKey); ok {
		return append(p[:len(p)-1], pathElement{pe.key, append(last.metadata, pe.metadata...)})
	}
	return append(p, pe)
}

func (p path) next() (*pathElement, path) {
	if len(p) == 0 {
		return nil, nil
	}
	pe := p[0]
	if _, ok := pe.key.(metadataOnlyPathKey); ok {
		return nil, nil
	}
	return &pe, p[1:]
}

func (p path) clone() path {
	c := make(path, len(p))
	copy(c, p)
	return c
}

func (p path) String() string {
	arr := jsonArray{}
	var meta jsonArray
	appendMetadata := func() {
		if len(meta) > 0 {
			arr = append(arr, meta)
			meta = nil
		}
	}
	for _, pe := range p {
		for _, m := range pe.metadata {
			meta = append(meta, jsonString(m.string()))
		}
		switch t := pe.key.(type) {
		case metadataOnlyPathKey:
		case stringPathKey:
			appendMetadata()
			arr = append(arr, jsonString(string(t)))
		case indexPathKey:
			appendMetadata()
			arr = append(arr, jsonNumber(int(t)))
		case setElementPathKey:
			appendMetadata()
			arr = append(arr, jsonObject{})
		case specificSetElementPathKey:
			appendMetadata()
			arr = append(arr, t.obj)
		}
	}
	return arr.Json()
}

func pathFromString(s string) (path, error) {
	n, err := ReadJsonString(s)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}
	arr, ok := n.(jsonArray)
	if !ok {
		return nil, fmt.Errorf("invalid path: got %T, want JSON list", n)
	}

	var ret path
	var meta []Metadata
	for _, e := range arr {
		switch t := e.(type) {
		case jsonArray:
			// TODO: parse metadata cleanly.
			for _, m := range t {
				if s, ok := m.(jsonString); ok {
					if string(s) == SET.string() {
						meta = append(meta, SET)
					}
					if string(s) == MULTISET.string() {
						meta = append(meta, MULTISET)
					}
					if string(s) == ASSOC_IN.string() {
						meta = append(meta, ASSOC_IN)
					}
				}
				// Ignore unrecognized metadata.
			}
		case jsonString:
			ret = ret.append(stringPathKey(string(t)), meta...)
			meta = nil
		case jsonNumber:
			ret = ret.append(indexPathKey(int(t)), meta...)
			meta = nil
		case jsonObject:
			// JSON object implies a set.
			if !checkMetadata(SET, meta) && !checkMetadata(MULTISET, meta) {
				meta = append(meta, SET)
			}

			if len(t.properties) == 0 {
				ret = ret.append(setElementPathKey{}, meta...)
			} else {
				ret = ret.append(specificSetElementPathKey{t}, meta...)
			}
			meta = nil
		}
	}
	return ret, nil
}
