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

type stringPathKey string
type indexPathKey int
type setElementPathKey struct{}
type specificSetElementPathKey struct{ obj jsonObject }

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

func (s stringPathKey) isPathKey()             {}
func (i indexPathKey) isPathKey()              {}
func (s setElementPathKey) isPathKey()         {}
func (s specificSetElementPathKey) isPathKey() {}

func (p path) append(k pathKey, m ...Metadata) path {
	return append(p, pathElement{k, m})
}

func (p path) clone() path {
	c := make(path, len(p))
	copy(c, p)
	return c
}

func (p path) String() string {
	arr := jsonArray{}
	for _, pe := range p {
		var meta jsonArray
		for _, m := range pe.metadata {
			meta = append(meta, jsonString(m.string()))
		}
		if len(meta) > 0 {
			arr = append(arr, meta)
		}
		switch t := pe.key.(type) {
		case stringPathKey:
			arr = append(arr, jsonString(string(t)))
		case indexPathKey:
			arr = append(arr, jsonNumber(int(t)))
		case setElementPathKey:
			arr = append(arr, jsonObject{})
		case specificSetElementPathKey:
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
