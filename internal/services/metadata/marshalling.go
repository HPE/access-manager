/*
 * SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
 */

package metadata

import (
	"encoding/json"
	"fmt"
	"github.com/hpe/access-manager/internal/services/common"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"strings"
)

/*
UnmarshalJSON extracts a metadata tree from JSON format. Note that the JSON
format for metadata follows the conventions of the protojson package.

Also, most datasets stored in JSON files don't have versions, uniques,
start times or end times. This doesn't usually matter since you can
put in quasi-realistic versions and uniques using tree.adjust(), but
it might catch one unaware when interesting values for those fields
seem to appear without cause.
*/
func (t *MetaTree) UnmarshalJSON(b []byte) error {
	var x struct {
		Path     string
		Meta     []json.RawMessage
		Children []*MetaTree
	}
	// the core structure of the tree is handled directly
	if err := json.Unmarshal(b, &x); err != nil {
		return err
	}
	t.Path = x.Path
	t.Children = x.Children

	// the annotations, however, need special treatment
	t.Meta = []*Annotation{}
	isLeaf := false
	for _, m := range x.Meta {
		annotation, leaf, err := UnmarshalAnnotation(m)
		if err != nil {
			return err
		}
		isLeaf = isLeaf || leaf
		if annotation.Unique == 0 {
			annotation.Unique = common.SafeUnique()
		}
		t.Meta = append(t.Meta, annotation)
	}
	if !isLeaf {
		t.Meta = append(t.Meta, &Annotation{Tag: "dir"})
	}
	return nil
}

func UnmarshalAnnotation(m []byte) (*Annotation, bool, error) {
	isLeaf := false

	var tag struct {
		Tag string `json:"tag"`
	}
	if err := json.Unmarshal(m, &tag); err != nil {
		return nil, false, err
	}
	var ax proto.Message
	switch tag.Tag {
	case "ace":
		ax = &ACE{}

	case "applied-role":
		ax = &AppliedRole{}

	case "role", "principal", "data":
		ax = &UserAnnotation{Tag: "leaf"}
		isLeaf = true

	default:
		ax = &UserAnnotation{}
	}

	annotation, err := UnMarshalNotable(m, ax)
	if err != nil {
		return nil, false, err
	}
	return annotation, isLeaf, nil
}

func UnMarshalNotable(m json.RawMessage, ax proto.Message) (*Annotation, error) {
	if err := protojson.Unmarshal(m, ax); err != nil {
		return nil, err
	}
	annotation, err := ax.(Notable).AsAnnotation()
	if err != nil {
		return nil, err
	}
	return annotation, nil
}

func (op *Operation) UnmarshalJSON(b []byte) error {
	z, ok := Operation_value[strings.Trim(strings.ToUpper(string(b)), `"`)]
	if !ok {
		return fmt.Errorf("invalid operation %q", b)
	}
	*op = Operation(z)
	return nil
}

/*
MarshalJSON produces a JSON form from a metadata tree. This includes tags on the
metadata so that the output can be read back in.

The only real gotcha if you read it back in is that all of the fake unique and
version values inserted by MetaTree.adjust() will explicitly be there. That
shouldn't be a problem, but it could be a surprise.
*/
func (t *MetaTree) MarshalJSON() ([]byte, error) {
	x := struct {
		Path     string
		Meta     []*Annotation
		Children []*MetaTree
	}{
		Path:     t.Path,
		Children: t.Children,
	}

	for _, m := range t.Meta {
		x.Meta = append(x.Meta, m)
	}
	return json.Marshal(x)
}

func (a *Annotation) MarshalJSON() ([]byte, error) {
	switch a.Tag {
	case "applied-role":
		r, err := a.AsAppliedRole()
		if err != nil {
			return nil, err
		}
		return json.Marshal(&r)

	case "ace":
		r, err := a.AsACE()
		if err != nil {
			return nil, err
		}
		return json.Marshal(&r)
	case "role", "principal", "data":
		b := *a
		b.Tag = "leaf"
		r, err := b.AsUserAnnotation()
		if err != nil {
			return nil, err
		}
		return json.Marshal(&r)
	default:
		r, err := a.AsUserAnnotation()
		if err != nil {
			return nil, err
		}
		return json.Marshal(&r)
	}
}

func (op Operation) MarshalJSON() ([]byte, error) {
	return []byte(`"` + Operation_name[int32(op)] + `"`), nil
}
