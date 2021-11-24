package jd

import (
	"fmt"
	"sort"
)

type jsonSet jsonArray

var _ JsonNode = jsonSet(nil)

func (s jsonSet) Json(metadata ...Metadata) string {
	return renderJson(s.raw(metadata))
}

func (s jsonSet) Yaml(metadata ...Metadata) string {
	return renderYaml(s.raw(metadata))
}

func (s jsonSet) raw(metadata []Metadata) interface{} {
	sMap := make(map[[8]byte]JsonNode)
	for _, n := range s {
		hc := n.hashCode(metadata)
		sMap[hc] = n
	}
	hashes := make(hashCodes, 0, len(sMap))
	for hc := range sMap {
		hashes = append(hashes, hc)
	}
	sort.Sort(hashes)
	set := make([]interface{}, 0, len(sMap))
	for _, hc := range hashes {
		set = append(set, sMap[hc].raw(metadata))
	}
	return set
}

func (s1 jsonSet) Equals(n JsonNode, metadata ...Metadata) bool {
	s2, ok := n.(jsonSet)
	if !ok {
		return false
	}
	if s1.hashCode(metadata) == s2.hashCode(metadata) {
		return true
	} else {
		return false
	}
}

func (s jsonSet) hashCode(metadata []Metadata) [8]byte {
	sMap := make(map[[8]byte]bool)
	for _, v := range s {
		v = dispatch(v, metadata)
		hc := v.hashCode(metadata)
		sMap[hc] = true
	}
	hashes := make(hashCodes, 0, len(sMap))
	for hc := range sMap {
		hashes = append(hashes, hc)
	}
	return hashes.combine()
}

func (s jsonSet) Diff(j JsonNode, metadata ...Metadata) Diff {
	return s.diff(j, make(path, 0), metadata)
}

func (s1 jsonSet) diff(n JsonNode, path path, metadata []Metadata) Diff {
	d := make(Diff, 0)
	s2, ok := n.(jsonSet)
	if !ok {
		// Different types
		e := DiffElement{
			Path:      path.clone(),
			OldValues: nodeList(s1),
			NewValues: nodeList(n),
		}
		return append(d, e)
	}
	s1Map, s1Hashes := diffMap(s1, metadata)
	s2Map, s2Hashes := diffMap(s2, metadata)

	e := DiffElement{
		Path:      path.append(setElementPathKey{}, metadata...).clone(),
		OldValues: nodeList(),
		NewValues: nodeList(),
	}
	for _, hc := range s1Hashes {
		n2, ok := s2Map[hc]
		if !ok {
			// Deleted value.
			e.OldValues = append(e.OldValues, s1Map[hc])
			continue
		}
		// Changed value.
		o1, isObject1 := s1Map[hc].(jsonObject)
		o2, isObject2 := n2.(jsonObject)
		if isObject1 && isObject2 {
			// Sub diff objects with same identity.
			p := path.append(specificSetElementPathKey{o1}, metadata...)
			subDiff := o1.diff(o2, p, metadata)
			for _, subElement := range subDiff {
				d = append(d, subElement)
			}
		}
		// else if isObject1 != isObject2: We have a hash collision between an object and a non-object, which is unlikely
		// else if !isObject1 && !isObject2: Non-objects are hashed by value, so they're equal and there's no diff.
	}
	for _, hc := range s2Hashes {
		_, ok := s1Map[hc]
		if !ok {
			// Added value.
			e.NewValues = append(e.NewValues, s2Map[hc])
		}
	}
	if len(e.OldValues) > 0 || len(e.NewValues) > 0 {
		d = append(d, e)
	}
	return d
}

func diffMap(s jsonSet, m []Metadata) (map[[8]byte]JsonNode, hashCodes) {
	res := make(map[[8]byte]JsonNode)
	for _, v := range s {
		var hc [8]byte
		if o, ok := v.(jsonObject); ok {
			// Hash objects by their identity.
			hc = o.ident(m)
		} else {
			// Everything else by full content.
			hc = v.hashCode(m)
		}
		res[hc] = v
	}
	var hashes hashCodes
	for hc := range res {
		hashes = append(hashes, hc)
	}
	sort.Sort(hashes)
	return res, hashes
}

func (s jsonSet) Patch(d Diff) (JsonNode, error) {
	return patchAll(s, d)
}

func (s jsonSet) patch(pathBehind, pathAhead path, oldValues, newValues []JsonNode) (JsonNode, error) {
	// Base case
	if len(pathAhead) == 0 {
		if len(oldValues) > 1 || len(newValues) > 1 {
			return patchErrNonSetDiff(oldValues, newValues, pathBehind)
		}
		oldValue := singleValue(oldValues)
		newValue := singleValue(newValues)
		if !s.Equals(oldValue) {
			return patchErrExpectValue(oldValue, s, pathBehind)
		}
		return newValue, nil
	}
	// Unrolled recursive case
	pe, rest := pathAhead[0], pathAhead[1:]
	var pathObject jsonObject
	switch t := pe.key.(type) {
	case setElementPathKey:
	case specificSetElementPathKey:
		pathObject = t.obj
	default:
		return nil, fmt.Errorf(
			"Invalid path element %v. Expected jsonObject.", t)
	}
	if len(rest) > 0 {
		// Recurse into a specific object.
		lookingFor := pathObject.ident(pe.metadata)
		for _, v := range s {
			if o, ok := v.(jsonObject); ok {
				id := o.pathIdent(pathObject, pe.metadata)
				if id == lookingFor {
					v.patch(append(pathBehind, pe), rest, oldValues, newValues)
					return s, nil
				}
			}
		}
		return nil, fmt.Errorf("Invalid diff. Expected object with id %v but found none", pathObject.Json(pe.metadata...))
	}
	// Patch set
	aMap := make(map[[8]byte]JsonNode)
	for _, v := range s {
		var hc [8]byte
		if o, ok := v.(jsonObject); ok {
			// Hash objects by their identitiy.
			hc = o.ident(pe.metadata)
		} else {
			// Everything else by full content.
			hc = v.hashCode(pe.metadata)
		}
		aMap[hc] = v
	}
	for _, v := range oldValues {
		var hc [8]byte
		if o, ok := v.(jsonObject); ok {
			// Find objects by their identitiy.
			hc = o.ident(pe.metadata)
		} else {
			// Everything else by full content.
			hc = v.hashCode(pe.metadata)
		}
		toDelete, ok := aMap[hc]
		if !ok {
			return nil, fmt.Errorf(
				"Invalid diff. Expected %v at %v but found nothing.",
				v.Json(pe.metadata...), pathBehind)
		}
		if !toDelete.Equals(v, pe.metadata...) {
			return nil, fmt.Errorf(
				"Invalid diff. Expected %v at %v but found %v.",
				v.Json(pe.metadata...), pathBehind, toDelete.Json(pe.metadata...))

		}
		delete(aMap, hc)
	}
	for _, v := range newValues {
		var hc [8]byte
		if o, ok := v.(jsonObject); ok {
			// Hash objects by their identitiy.
			hc = o.ident(pe.metadata)
		} else {
			// Everything else by full content.
			hc = v.hashCode(pe.metadata)
		}
		aMap[hc] = v
	}
	hashes := make(hashCodes, 0, len(aMap))
	for hc := range aMap {
		hashes = append(hashes, hc)
	}
	sort.Sort(hashes)
	newValue := make(jsonSet, 0, len(aMap))
	for _, hc := range hashes {
		newValue = append(newValue, aMap[hc])
	}
	return newValue, nil
}
