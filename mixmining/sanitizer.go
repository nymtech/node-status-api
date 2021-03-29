// Copyright 2020 Nym Technologies SA
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mixmining

import (
	"fmt"
	"github.com/microcosm-cc/bluemonday"
	"github.com/nymtech/nym/validator/nym/directory/models"
	"os"
	"reflect"
)


// GenericSanitizer sanitizes untrusted data of any type. It mutates its arguments in place.
type GenericSanitizer interface {
	Sanitize(input interface{})
}

type genericSanitizer struct {
	policy *bluemonday.Policy
}

// NewSanitizer returns a new input sanitizer for all presence-related things
func NewGenericSanitizer(policy *bluemonday.Policy) GenericSanitizer {
	return genericSanitizer{
		policy: policy,
	}
}

func (s genericSanitizer) sanitizeStruct(v reflect.Value) {
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		kind := field.Kind()

		switch kind {
		case reflect.String:
			if !field.CanSet() {
				fmt.Printf("wtf can't set %v (type: %v)", field, kind)
				continue
			}
			field.SetString(s.policy.Sanitize(field.String()))
		case reflect.Struct:
			s.Sanitize(v.Field(i).Addr().Interface())
		case reflect.Int64:
		case reflect.Uint:
			continue
		default:
			fmt.Fprintf(os.Stderr, "tried to sanitize unknown type %+v\n", kind)
		}
	}
}

func (s genericSanitizer) Sanitize(input interface{}) {
	v := reflect.ValueOf(input)
	v = reflect.Indirect(v)

	inputKind := v.Kind()
	switch inputKind {
	case reflect.String:
		v.SetString(s.policy.Sanitize(v.String()))
	case reflect.Struct:
		s.sanitizeStruct(v)
	default:
		fmt.Fprintf(os.Stderr, "tried to sanitize unknown type %+v\n", inputKind)
	}

}

// BatchSanitizer sanitizes untrusted batch mixmining data. It should be used in
// controllers to wipe out any questionable input at our application's front
// door.
type BatchSanitizer interface {
	Sanitize(input models.BatchMixStatus) models.BatchMixStatus
}

type batchSanitizer struct {
	sanitizer sanitizer
}

// NewBatchSanitizer returns a new input sanitizer for metrics
func NewBatchSanitizer(policy *bluemonday.Policy) BatchSanitizer {
	return batchSanitizer{
		sanitizer: sanitizer{
			policy: policy,
		},
	}
}

func (s batchSanitizer) Sanitize(input models.BatchMixStatus) models.BatchMixStatus {
	for i := range input.Status {
		input.Status[i] = s.sanitizer.Sanitize(input.Status[i])
	}
	return input
}

// Sanitizer sanitizes untrusted mixmining data. It should be used in
// controllers to wipe out any questionable input at our application's front
// door.
type Sanitizer interface {
	Sanitize(input models.MixStatus) models.MixStatus
}

type sanitizer struct {
	policy *bluemonday.Policy
}

// NewSanitizer returns a new input sanitizer for metrics
func NewSanitizer(policy *bluemonday.Policy) Sanitizer {
	return sanitizer{
		policy: policy,
	}
}

func (s sanitizer) Sanitize(input models.MixStatus) models.MixStatus {
	sanitized := newMeasurement()

	sanitized.PubKey = s.policy.Sanitize(input.PubKey)
	sanitized.IPVersion = s.policy.Sanitize(input.IPVersion)
	sanitized.Up = input.Up
	return sanitized
}

func newMeasurement() models.MixStatus {
	booltrue := true
	return models.MixStatus{
		PubKey:    "",
		IPVersion: "",
		Up:        &booltrue,
	}
}
