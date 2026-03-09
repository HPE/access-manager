/*
 * SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
 */

package metadata

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	Read      = Operation_READ  // read from a dataset
	Write     = Operation_WRITE // write to a dataset, create children of a directory
	View      = Operation_VIEW  // is an entity visible (could be a dataset, role, User = Operation_user          // can permissions on an entity be modified?
	Admin     = Operation_ADMIN
	UseRole   = Operation_USEROLE   // can a role be used in an ACL?
	ApplyRole = Operation_APPLYROLE // can a role be added to a principal?
	VouchFor  = Operation_VOUCHFOR  // can a plugin vouch for an identity?
	Invalid   = Operation_INVALID
)

type Notable interface {
	AsAnnotation() (*Annotation, error)
}

// Annotation serves as the wrapper for all annotations and contains
// the common fields. This is a plain Go object so that we can have
// a cached value for the unmarshalled annotation itself. There is a
// parallel structure called AnnotationWrapper in the protobuf that
// is actually used as the format for storing annotations.
type Annotation struct {
	Unique      int64         `json:",string"` //  unique-ifier to avoid collisions for annotations on a metadata node
	Version     int64         `json:",string"` // version for safe updates
	Global      int64         `json:",string"` // the global revision number for last modification
	StartMillis int64         `json:",string"` // what clock time did this annotation take effect
	EndMillis   int64         `json:",string"` // when does this annotation lose effect
	Tag         string        // hint for the interpretation of the value
	Raw         *anypb.Any    // any data specific to the annotation goes here in marshalled form
	Value       proto.Message // and here in unmarshalled form
}

// instantiate forces the Value field of an annotation to be instantiated,
// unmarshalling it from the Raw field if necessary. This code understands
// applied roles and ACEs natively so `v` can be `nil` for these. All other data
// types will require a target of the instantiation to be passed in via the
// argument `v`.
func (a *Annotation) instantiate(v proto.Message) error {
	if a.Value == nil {
		if a.Raw == nil {
			return nil
		}
		if v == nil {
			switch a.Tag {
			case "ace":
				v = &ACEPersist{}
			case "applied-role":
				v = &RolePersist{}
			default:
				return nil
			}
		}
		if err := anypb.UnmarshalTo(a.Raw, v, proto.UnmarshalOptions{}); err != nil {
			return err
		}
		a.Value = v
	}
	return nil
}

func Op(op string) Operation {
	r, ok := Operation_value[strings.ToUpper(op)]
	if !ok {
		return Operation_INVALID
	}
	return Operation(r)
}

func (r *AppliedRole) AsAnnotation() (*Annotation, error) {
	ann := Annotation{
		Unique:      r.Unique,
		Version:     r.Version,
		StartMillis: r.StartMillis,
		EndMillis:   r.EndMillis,
		Tag:         r.Tag,
	}
	v := RolePersist{
		Path: r.Role,
	}
	ann.Value = &v
	ann.Raw = &anypb.Any{}
	if err := anypb.MarshalFrom(ann.Raw, &v, proto.MarshalOptions{}); err != nil {
		return nil, fmt.Errorf("failed to convert to annotation: %w", err)
	}
	return &ann, nil
}

func (r *Annotation) AsAnnotation() (*Annotation, error) {
	return r, nil
}

func (r *ACE) AsAnnotation() (*Annotation, error) {
	ann := Annotation{
		Unique:      r.Unique,
		Version:     r.Version,
		StartMillis: r.StartMillis,
		EndMillis:   r.EndMillis,
		Tag:         r.Tag,
	}
	acls := []*ACLPersist{}
	for _, ax := range r.Acls {
		acls = append(acls, &ACLPersist{
			Roles: ax.Roles,
		})
	}
	v := ACEPersist{
		Op:    r.Op,
		Local: r.Local,
		Acls:  acls,
	}
	ann.Value = &v
	ann.Raw = &anypb.Any{}
	if err := anypb.MarshalFrom(ann.Raw, &v, proto.MarshalOptions{}); err != nil {
		return nil, fmt.Errorf("failed to convert to annotation: %w", err)
	}
	return &ann, nil
}

func UserAnnotationString(tag, value string) (*Annotation, error) {
	ua, err := (&UserAnnotation{
		Tag:  tag,
		Data: value,
	}).AsAnnotation()
	if err != nil {
		return nil, err
	}
	return ua, nil
}

func (r *UserAnnotation) AsAnnotation() (*Annotation, error) {
	ann := Annotation{
		Unique:      r.Unique,
		Version:     r.Version,
		Global:      0,
		StartMillis: r.StartMillis,
		EndMillis:   r.EndMillis,
		Tag:         r.Tag,
	}
	z := TaggedString{
		Tag:   r.Tag,
		Value: r.Data,
	}
	v, err := anypb.New(&z)
	if err != nil {
		return nil, err
	}
	ann.Raw = v
	return &ann, nil
}

func (r *UserAnnotation) AsString(tag string) (string, error) {
	if r.Tag != tag {
		return "", fmt.Errorf("expected tag %q but got %q", tag, r.Tag)
	}
	return r.Data, nil
}

func (ann *Annotation) AsAppliedRole() (*AppliedRole, error) {
	if err := ann.instantiate(nil); err != nil {
		return nil, err
	}
	if ann.Tag != "applied-role" {
		return nil, fmt.Errorf("annotation is not an applied role")
	}
	r := ann.Value.(*RolePersist)
	return &AppliedRole{
		Role:        r.Path,
		Tag:         "applied-role",
		Unique:      ann.Unique,
		Version:     ann.Version,
		StartMillis: ann.StartMillis,
		EndMillis:   ann.EndMillis,
	}, nil
}

func (ann *Annotation) AsACE() (*ACE, error) {
	if ann.Tag != "ace" {
		return nil, fmt.Errorf("annotation is not an ace")
	}
	if err := ann.instantiate(nil); err != nil {
		return nil, err
	}
	r := ann.Value.(*ACEPersist)
	acls := []*ACL{}
	for _, rx := range r.Acls {
		acls = append(acls, &ACL{
			Roles: rx.Roles,
		})
	}
	return &ACE{
		Op:          r.Op,
		Local:       r.Local,
		Acls:        acls,
		Tag:         "ace",
		Unique:      ann.Unique,
		Version:     ann.Version,
		StartMillis: ann.StartMillis,
		EndMillis:   ann.EndMillis,
	}, nil
}

func (ann *Annotation) AsUserAnnotation() (*UserAnnotation, error) {
	var s TaggedString
	if err := anypb.UnmarshalTo(ann.Raw, &s, proto.UnmarshalOptions{}); err != nil {
		return nil, err
	}
	if s.Tag != ann.Tag {
		return nil, fmt.Errorf("expected tag %q but got %q", ann.Tag, s.Tag)
	}
	r := UserAnnotation{
		Data:        s.Value,
		Tag:         ann.Tag,
		Unique:      ann.Unique,
		Version:     ann.Version,
		StartMillis: ann.StartMillis,
		EndMillis:   ann.EndMillis,
	}
	return &r, nil
}
