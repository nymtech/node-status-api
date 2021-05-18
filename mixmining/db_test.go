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
	"github.com/BorisBorshevsky/timemock"
	"github.com/nymtech/node-status-api/mixmining/fixtures"
	"github.com/nymtech/node-status-api/models"
	. "github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
	"time"
)

var _ = Describe("The mixmining db", func() {
	Describe("Constructing a NewDb", func() {
		Context("a new db", func() {
			It("should have no mixmining statuses", func() {
				db := NewDb(true)
				db.orm.Exec("DELETE FROM persisted_mix_statuses")
				assert.Len(GinkgoT(), db.ListMixStatus("foo", 5), 0)
			})
		})
	})

	Describe("adding and retrieving measurements", func() {
		Context("a new db", func() {
			It("should add measurements to the db, with a timestamp, and be able to retrieve them afterwards", func() {
				db := NewDb(true)
				db.orm.Exec("DELETE FROM persisted_mix_statuses")
				status := fixtures.GoodPersistedMixStatus()

				// add one
				db.AddMixStatus(status)
				measurements := db.ListMixStatus(status.PubKey, 5)
				assert.Len(GinkgoT(), measurements, 1)
				assert.Equal(GinkgoT(), status, measurements[0])

				// add another
				db.AddMixStatus(status)
				measurements = db.ListMixStatus(status.PubKey, 5)
				assert.Len(GinkgoT(), measurements, 2)
				assert.Equal(GinkgoT(), status, measurements[0])
				assert.Equal(GinkgoT(), status, measurements[1])
			})
		})
	})

	Describe("listing mix statuses within a date range", func() {
		Context("for an empty db", func() {
			It("should return an empty slice", func() {
				db := NewDb(true)
				db.orm.Exec("DELETE FROM persisted_mix_statuses")
				assert.Len(GinkgoT(), db.ListMixStatusDateRange("foo", "6", 1, 1), 0)
			})
		})
		Context("when one status exists in the range and one outside", func() {
			It("should return only the status within the range", func() {
				db := NewDb(true)
				db.orm.Exec("DELETE FROM persisted_mix_statuses")
				data := fixtures.GoodMixStatus()
				statusInRange := models.PersistedMixStatus{
					MixStatus: data,
					Timestamp: 500,
				}
				statusOutOfRange := models.PersistedMixStatus{
					MixStatus: data,
					Timestamp: 1000,
				}
				db.AddMixStatus(statusInRange)
				db.AddMixStatus(statusOutOfRange)

				result := db.ListMixStatusDateRange(data.PubKey, "6", 0, 500)
				assert.Len(GinkgoT(), result, 1)
				assert.Equal(GinkgoT(), statusInRange, result[0])
			})
		})
		Context("when one Ipv4 status exists in the range and one outside, with an IPv6 status also in range, when searching for IPv4", func() {
			It("should return only the status within the range", func() {
				db := NewDb(true)
				db.orm.Exec("DELETE FROM persisted_mix_statuses")
				ip4data := fixtures.GoodMixStatus()
				ip4data.IPVersion = "4"

				ip6data := fixtures.GoodMixStatus()
				ip6data.IPVersion = "6"
				ip4statusInRange := models.PersistedMixStatus{
					MixStatus: ip4data,
					Timestamp: 500,
				}
				ip6statusInRange := models.PersistedMixStatus{
					MixStatus: ip6data,
					Timestamp: 500,
				}
				ip4statusOutOfRange := models.PersistedMixStatus{
					MixStatus: ip4data,
					Timestamp: 1000,
				}
				db.AddMixStatus(ip4statusInRange)
				db.AddMixStatus(ip6statusInRange)
				db.AddMixStatus(ip4statusOutOfRange)

				result := db.ListMixStatusDateRange(ip4statusInRange.PubKey, "4", 0, 500)
				assert.Len(GinkgoT(), result, 1)
				assert.Equal(GinkgoT(), ip4statusInRange, result[0])
			})
		})
	})

	Describe("listing mix statuses with a limit", func() {
		Context("for an empty db", func() {
			It("should return an empty slice", func() {
				db := NewDb(true)
				defer db.orm.Exec("DELETE FROM persisted_mix_statuses")
				assert.Len(GinkgoT(), db.ListMixStatus("foo", 5), 0)
			})
		})
	})

	Describe("saving a mix status report", func() {
		Context("for an empty db", func() {
			It("should save and reload the report", func() {
				db := NewDb(true)
				db.orm.Exec("DELETE FROM mix_status_reports")
				newReport := models.MixStatusReport{
					PubKey:           "key",
					MostRecentIPV4:   true,
					Last5MinutesIPV4: 5,
					LastHourIPV4:     10,
					LastDayIPV4:      15,
					MostRecentIPV6:   false,
					Last5MinutesIPV6: 30,
					LastHourIPV6:     40,
					LastDayIPV6:      50,
				}
				db.SaveMixStatusReport(newReport)
				saved := db.LoadMixReport(newReport.PubKey)
				assert.Equal(GinkgoT(), newReport, saved)
			})
		})
		Context("when saving a second time", func() {
			It("should re-save the original report, and not make a second copy", func() {
				db := NewDb(true)
				db.orm.Exec("DELETE FROM mix_status_reports")

				newReport := models.MixStatusReport{
					PubKey:           "key",
					MostRecentIPV4:   true,
					Last5MinutesIPV4: 5,
					LastHourIPV4:     10,
					LastDayIPV4:      15,
					MostRecentIPV6:   false,
					Last5MinutesIPV6: 30,
					LastHourIPV6:     40,
					LastDayIPV6:      50,
				}
				db.SaveMixStatusReport(newReport)

				var firstCount int64
				db.orm.Model(&models.MixStatusReport{}).Where("pub_key = ?", "key").Count(&firstCount)
				assert.Equal(GinkgoT(), int64(1), firstCount)

				report := db.LoadMixReport("key")
				report.Last5MinutesIPV4 = 666

				db.SaveMixStatusReport(report)

				var secondCount int64
				db.orm.Model(&models.MixStatusReport{}).Where("pub_key = ?", "key").Count(&secondCount)
				assert.Equal(GinkgoT(), int64(1), secondCount)

				reloadedReport := db.LoadMixReport("key")
				assert.Equal(GinkgoT(), 666, reloadedReport.Last5MinutesIPV4)
			})
		})
	})

	Describe("Getting active nodes", func() {
		It("Returns list of public keys of nodes seen in specified time period without duplicates", func() {
			now := timemock.Now()

			db := NewDb(true)

			status1 := models.PersistedMixStatus{
				MixStatus: models.MixStatus{
					PubKey:           "aaa",
				},
				Timestamp: now.UnixNano(),
			}

			status2 := models.PersistedMixStatus{
				MixStatus: models.MixStatus{
					PubKey:           "bbb",
				},
				Timestamp: now.UnixNano(),
			}

			status3 := models.PersistedMixStatus{
				MixStatus: models.MixStatus{
					PubKey:           "ccc",
				},
				Timestamp: now.UnixNano(),
			}

			status4Duplicate := models.PersistedMixStatus{
				MixStatus: models.MixStatus{
					PubKey:           "ccc",
				},
				Timestamp: timemock.Now().UnixNano(),
			}

			db.AddMixStatus(status1)
			db.AddMixStatus(status2)
			db.AddMixStatus(status3)
			db.AddMixStatus(status4Duplicate)

			dayAgo := now.Add(time.Duration(-1) * time.Hour * 24).UnixNano()
			active := db.GetActiveMixes(dayAgo)

			assert.Equal(GinkgoT(), active, []string{"aaa", "bbb", "ccc"})
		})
	})
})
