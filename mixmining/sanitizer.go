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
	"os"
	"reflect"

	"github.com/microcosm-cc/bluemonday"
	"github.com/nymtech/node-status-api/models"
)

// GenericSanitizer sanitizes untrusted data of any type. It mutates its arguments in place.
type GenericSanitizer interface {
	Sanitize(input interface{})
}

type genericSanitizer struct {
	policy *bluemonday.Policy
}

// NewMixStatusSanitizer returns a new input mixStatusSanitizer for all presence-related things
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

// BatchMixSanitizer sanitizes untrusted batch mixmining data. It should be used in
// controllers to wipe out any questionable input at our application's front
// door.
type BatchMixSanitizer interface {
	Sanitize(input models.BatchMixStatus) models.BatchMixStatus
}

type batchMixSanitizer struct {
	sanitizer mixStatusSanitizer
}

// NewBatchMixSanitizer returns a new input mixStatusSanitizer for metrics
func NewBatchMixSanitizer(policy *bluemonday.Policy) BatchMixSanitizer {
	return batchMixSanitizer{
		sanitizer: mixStatusSanitizer{
			policy: policy,
		},
	}
}

func (s batchMixSanitizer) Sanitize(input models.BatchMixStatus) models.BatchMixStatus {
	for i := range input.Status {
		input.Status[i] = s.sanitizer.Sanitize(input.Status[i])
	}
	return input
}

// BatchGatewaySanitizer sanitizes untrusted batch mixmining data. It should be used in
// controllers to wipe out any questionable input at our application's front
// door.
type BatchGatewaySanitizer interface {
	Sanitize(input models.BatchGatewayStatus) models.BatchGatewayStatus
}

type batchGatewaySanitizer struct {
	sanitizer gatewayStatusSanitizer
}

// NewBatchGatewaySanitizer returns a new input mixStatusSanitizer for metrics
func NewBatchGatewaySanitizer(policy *bluemonday.Policy) BatchGatewaySanitizer {
	return batchGatewaySanitizer{
		sanitizer: gatewayStatusSanitizer{
			policy: policy,
		},
	}
}

func (s batchGatewaySanitizer) Sanitize(input models.BatchGatewayStatus) models.BatchGatewayStatus {
	for i := range input.Status {
		input.Status[i] = s.sanitizer.Sanitize(input.Status[i])
	}
	return input
}

// MixStatusSanitizer sanitizes untrusted mixmining data. It should be used in
// controllers to wipe out any questionable input at our application's front
// door.
type MixStatusSanitizer interface {
	Sanitize(input models.MixStatus) models.MixStatus
}

type mixStatusSanitizer struct {
	policy *bluemonday.Policy
}

// NewMixStatusSanitizer returns a new input mixStatusSanitizer for metrics
func NewMixStatusSanitizer(policy *bluemonday.Policy) MixStatusSanitizer {
	return mixStatusSanitizer{
		policy: policy,
	}
}

func (s mixStatusSanitizer) Sanitize(input models.MixStatus) models.MixStatus {
	sanitized := newMixMeasurement()

	sanitized.PubKey = s.policy.Sanitize(input.PubKey)
	sanitized.Owner = s.policy.Sanitize(input.Owner)
	sanitized.IPVersion = s.policy.Sanitize(input.IPVersion)
	sanitized.Up = input.Up
	return sanitized
}

func newMixMeasurement() models.MixStatus {
	booltrue := true
	return models.MixStatus{
		PubKey:    "",
		Owner: "",
		IPVersion: "",
		Up:        &booltrue,
	}
}


// GatewayStatusSanitizer sanitizes untrusted mixmining data. It should be used in
// controllers to wipe out any questionable input at our application's front
// door.
type GatewayStatusSanitizer interface {
	Sanitize(input models.GatewayStatus) models.GatewayStatus
}

type gatewayStatusSanitizer struct {
	policy *bluemonday.Policy
}

// NewGatewayStatusSanitizer returns a new input mixStatusSanitizer for metrics
func NewGatewayStatusSanitizer(policy *bluemonday.Policy) GatewayStatusSanitizer {
	return gatewayStatusSanitizer{
		policy: policy,
	}
}

func (s gatewayStatusSanitizer) Sanitize(input models.GatewayStatus) models.GatewayStatus {
	sanitized := newGatewayMeasurement()

	sanitized.PubKey = s.policy.Sanitize(input.PubKey)
	sanitized.Owner = s.policy.Sanitize(input.Owner)
	sanitized.IPVersion = s.policy.Sanitize(input.IPVersion)
	sanitized.Up = input.Up
	return sanitized
}

func newGatewayMeasurement() models.GatewayStatus {
	booltrue := true
	return models.GatewayStatus{
		PubKey:    "",
		Owner: "",
		IPVersion: "",
		Up:        &booltrue,
	}
}
