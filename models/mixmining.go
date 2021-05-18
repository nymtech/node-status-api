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

package models

import (
	_ "github.com/jinzhu/gorm"
)

// MixStatus indicates whether a given node is up or down, as reported by a Nym monitor node.
// The 'Up' field is pretty annoying. Gin and other HTTP routers ignore incoming json "false" values,
// so making it a pointer works. This necessitates crapification of the Up-related code, as you can't
// do `*true` or `&true`, you need a variable to point to or dereference. This is why you'll see e.g.
// things like `booltrue := true`, `&booltrue` in the codebase. Maybe there's a more elegant way to
// achieve that which a bigger gopher could clean up.
type MixStatus struct {
	PubKey    string `json:"pubKey" binding:"required" gorm:"index:mix_status_index"`
	Owner     string `json:"owner" binding:"required" gorm:"index:mix_status_index"`
	IPVersion string `json:"ipVersion" binding:"required" gorm:"index:mix_status_index"`
	Up        *bool  `json:"up" binding:"required"`
}

type GatewayStatus struct {
	PubKey    string `json:"pubKey" binding:"required" gorm:"index:gateway_status_index"`
	Owner     string `json:"owner" binding:"required" gorm:"index:gateway_status_index"`
	IPVersion string `json:"ipVersion" binding:"required" gorm:"index:gateway_status_index"`
	Up        *bool  `json:"up" binding:"required"`
}

// PersistedMixStatus is a saved MixStatus with a timestamp recording when it
// was seen by the directory server. It can be used to build visualizations of
// mixnode uptime.
type PersistedMixStatus struct {
	MixStatus
	Timestamp int64 `json:"timestamp" binding:"required" gorm:"index:mix_status_index,sort:desc"`
}

type PersistedGatewayStatus struct {
	GatewayStatus
	Timestamp int64 `json:"timestamp" binding:"required" gorm:"index:gateway_status_index,sort:desc"`
}

// MixStatusReport gives a quick view of mixnode uptime performance
type MixStatusReport struct {
	PubKey           string `json:"pubKey" binding:"required" gorm:"primaryKey;unique"`
	Owner            string `json:"owner" binding:"required" binding:"required"`
	MostRecentIPV4   bool   `json:"mostRecentIPV4" binding:"required"`
	Last5MinutesIPV4 int    `json:"last5MinutesIPV4" binding:"required"`
	LastHourIPV4     int    `json:"lastHourIPV4" binding:"required"`
	LastDayIPV4      int    `json:"lastDayIPV4" binding:"required"`
	MostRecentIPV6   bool   `json:"mostRecentIPV6" binding:"required"`
	Last5MinutesIPV6 int    `json:"last5MinutesIPV6" binding:"required"`
	LastHourIPV6     int    `json:"lastHourIPV6" binding:"required"`
	LastDayIPV6      int    `json:"lastDayIPV6" binding:"required"`
}

type GatewayStatusReport struct {
	PubKey           string `json:"pubKey" binding:"required" gorm:"primaryKey;unique"`
	Owner            string `json:"owner" binding:"required" binding:"required"`
	MostRecentIPV4   bool   `json:"mostRecentIPV4" binding:"required"`
	Last5MinutesIPV4 int    `json:"last5MinutesIPV4" binding:"required"`
	LastHourIPV4     int    `json:"lastHourIPV4" binding:"required"`
	LastDayIPV4      int    `json:"lastDayIPV4" binding:"required"`
	MostRecentIPV6   bool   `json:"mostRecentIPV6" binding:"required"`
	Last5MinutesIPV6 int    `json:"last5MinutesIPV6" binding:"required"`
	LastHourIPV6     int    `json:"lastHourIPV6" binding:"required"`
	LastDayIPV6      int    `json:"lastDayIPV6" binding:"required"`
}

// BatchMixStatus allows to indicate whether given set of nodes is up or down, as reported by a Nym monitor node.
type BatchMixStatus struct {
	Status []MixStatus `json:"status" binding:"required"`
}

type BatchGatewayStatus struct {
	Status []GatewayStatus `json:"status" binding:"required"`
}

// BatchMixStatusReport gives a quick view of network uptime performance
type BatchMixStatusReport struct {
	Report []MixStatusReport `json:"report" binding:"required"`
}

type BatchGatewayStatusReport struct {
	Report []GatewayStatusReport `json:"report" binding:"required"`
}

func (report *BatchMixStatusReport) SplitToChunks(chunkSize int) []BatchMixStatusReport {
	reportDataCopy := make([]MixStatusReport, len(report.Report))
	copy(reportDataCopy, report.Report)

	var chunks [][]MixStatusReport
	for chunkSize < len(reportDataCopy) {
		reportDataCopy, chunks = reportDataCopy[chunkSize:], append(chunks, reportDataCopy[0:chunkSize:chunkSize])
	}

	chunks = append(chunks, reportDataCopy)


	newReports := make([]BatchMixStatusReport, len(chunks))
	for i, chunk := range chunks {
		newReports[i] = BatchMixStatusReport{Report: chunk}
	}

	return newReports
}

func (report *BatchGatewayStatusReport) SplitToChunks(chunkSize int) []BatchGatewayStatusReport {
	reportDataCopy := make([]GatewayStatusReport, len(report.Report))
	copy(reportDataCopy, report.Report)

	var chunks [][]GatewayStatusReport
	for chunkSize < len(reportDataCopy) {
		reportDataCopy, chunks = reportDataCopy[chunkSize:], append(chunks, reportDataCopy[0:chunkSize:chunkSize])
	}

	chunks = append(chunks, reportDataCopy)


	newReports := make([]BatchGatewayStatusReport, len(chunks))
	for i, chunk := range chunks {
		newReports[i] = BatchGatewayStatusReport{Report: chunk}
	}

	return newReports
}