package jd

import (
	"bytes"
	"encoding/binary"
)

type jsonNumber float64

var _ JsonNode = jsonNumber(0)

func (n jsonNumber) Json(metadata ...Metadata) string {
	return renderJson(n.raw(metadata))
}

func (n jsonNumber) Yaml(metadata ...Metadata) string {
	return renderYaml(n.raw(metadata))
}

func (n jsonNumber) raw(_ []Metadata) interface{} {
	return float64(n)
}

func (n1 jsonNumber) Equals(node JsonNode, metadata ...Metadata) bool {
	n2, ok := node.(jsonNumber)
	if !ok {
		return false
	}
	if n1 != n2 {
		return false
	}
	return true
}

func (n jsonNumber) hashCode(metadata []Metadata) [8]byte {
	a := make([]byte, 0, 8)
	b := bytes.NewBuffer(a)
	binary.Write(b, binary.LittleEndian, n)
	return hash(b.Bytes())
}

func (n jsonNumber) Diff(node JsonNode, metadata ...Metadata) Diff {
	mask := getMask(metadata)
	return n.diff(node, make(path, 0), metadata, mask)
}

func (n jsonNumber) diff(node JsonNode, path path, metadata []Metadata, mask Mask) Diff {
	d := make(Diff, 0)
	if n.Equals(node) {
		return d
	}
	e := DiffElement{
		Path:      path.clone(),
		OldValues: nodeList(n),
		NewValues: nodeList(node),
	}
	return append(d, e)
}

func (n jsonNumber) Patch(d Diff) (JsonNode, error) {
	return patchAll(n, d)
}

func (n jsonNumber) patch(pathBehind, pathAhead path, oldValues, newValues []JsonNode) (JsonNode, error) {
	if len(pathAhead) != 0 {
		return patchErrExpectColl(n, pathAhead[0])
	}
	if len(oldValues) > 1 || len(newValues) > 1 {
		return patchErrNonSetDiff(oldValues, newValues, pathBehind)
	}
	oldValue := singleValue(oldValues)
	newValue := singleValue(newValues)
	if !n.Equals(oldValue) {
		return patchErrExpectValue(oldValue, n, pathBehind)
	}
	return newValue, nil
}
