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
	"time"

	"github.com/nymtech/node-status-api/mixmining/fixtures"
	"github.com/nymtech/node-status-api/models"
	. "github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
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
				saved := db.LoadReport(newReport.PubKey)
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

				report := db.LoadReport("key")
				report.Last5MinutesIPV4 = 666

				db.SaveMixStatusReport(report)

				var secondCount int64
				db.orm.Model(&models.MixStatusReport{}).Where("pub_key = ?", "key").Count(&secondCount)
				assert.Equal(GinkgoT(), int64(1), secondCount)

				reloadedReport := db.LoadReport("key")
				assert.Equal(GinkgoT(), 666, reloadedReport.Last5MinutesIPV4)
			})
		})
	})

	Describe("Registering mix node", func() {
		Context("For the first time", func() {
			It("should add the entry, with timestamp and initial reputation, to database", func() {
				db := NewDb(true)
				all := db.allRegisteredMixes()
				assert.Len(GinkgoT(), all, 0)

				mix := fixtures.GoodRegisteredMix()
				startTime := time.Now()
				db.RegisterMix(mix)
				endTime := time.Now()

				all = db.allRegisteredMixes()
				assert.Len(GinkgoT(), all, 1)
				assert.True(GinkgoT(), all[0].RegistrationTime >= startTime.UnixNano())
				assert.True(GinkgoT(), all[0].RegistrationTime <= endTime.UnixNano())

				// this is just so the comparison is easier
				all[0].RegistrationTime = 0
				assert.Equal(GinkgoT(), mix, all[0])
			})
		})
		Context("For second time", func() {
			It("should overwrite the existing entry without making a new one", func() {
				db := NewDb(true)
				all := db.allRegisteredMixes()
				assert.Len(GinkgoT(), all, 0)

				initialMix := fixtures.GoodRegisteredMix()
				updatedInitialMix := fixtures.GoodRegisteredMix()
				updatedInitialMix.Location = "New Foomplandia"
				updatedInitialMix.MixHost = "100.100.100.100:1789"

				db.RegisterMix(initialMix)
				all = db.allRegisteredMixes()
				initRegTime := all[0].RegistrationTime
				assert.Len(GinkgoT(), all, 1)

				db.RegisterMix(updatedInitialMix)
				all = db.allRegisteredMixes()

				assert.Len(GinkgoT(), all, 1)
				// since we 'registered' again we should get new registration time
				assert.True(GinkgoT(), all[0].RegistrationTime > initRegTime)

				// this is just so the comparison is easier
				all[0].RegistrationTime = 0
				assert.Equal(GinkgoT(), updatedInitialMix, all[0])
			})
		})

		Context("Multiple with different identity", func() {
			It("Should not overwrite each other", func() {
				db := NewDb(true)
				all := db.allRegisteredMixes()
				assert.Len(GinkgoT(), all, 0)

				initialMix1 := fixtures.GoodRegisteredMix()
				initialMix2 := fixtures.GoodRegisteredMix()
				initialMix2.IdentityKey = "NewID"

				db.RegisterMix(initialMix1)
				db.RegisterMix(initialMix2)
				all = db.allRegisteredMixes()
				assert.Len(GinkgoT(), all, 2)
			})
		})
	})

	Describe("Removing mix node", func() {
		Context("If it exists", func() {
			It("Should get rid of it", func() {
				db := NewDb(true)
				all := db.allRegisteredMixes()
				assert.Len(GinkgoT(), all, 0)

				mix := fixtures.GoodRegisteredMix()
				db.RegisterMix(mix)
				wasRemoved := db.UnregisterNode(mix.IdentityKey)
				assert.True(GinkgoT(), wasRemoved)

				all = db.allRegisteredMixes()
				assert.Len(GinkgoT(), all, 0)
			})
		})

		Context("If it doesn't exist", func() {
			It("Shouldn't do anything", func() {
				db := NewDb(true)
				all := db.allRegisteredMixes()
				assert.Len(GinkgoT(), all, 0)

				wasRemoved := db.UnregisterNode("foomp")
				assert.False(GinkgoT(), wasRemoved)

				all = db.allRegisteredMixes()
				assert.Len(GinkgoT(), all, 0)
			})
		})
	})

	Describe("Registering gateway", func() {
		Context("For the first time", func() {
			It("should add the entry, with timestamp and initial reputation, to database", func() {
				db := NewDb(true)
				all := db.allRegisteredGateways()
				assert.Len(GinkgoT(), all, 0)

				gateway := fixtures.GoodRegisteredGateway()
				startTime := time.Now()
				db.RegisterGateway(gateway)
				endTime := time.Now()

				all = db.allRegisteredGateways()
				assert.Len(GinkgoT(), all, 1)
				assert.True(GinkgoT(), all[0].RegistrationTime >= startTime.UnixNano())
				assert.True(GinkgoT(), all[0].RegistrationTime <= endTime.UnixNano())

				// this is just so the comparison is easier
				all[0].RegistrationTime = 0
				assert.Equal(GinkgoT(), gateway, all[0])
			})
		})
		Context("For second time", func() {
			It("should overwrite the existing entry without making a new one", func() {
				db := NewDb(true)
				all := db.allRegisteredGateways()
				assert.Len(GinkgoT(), all, 0)

				initialGateway := fixtures.GoodRegisteredGateway()
				updatedInitialGateway := fixtures.GoodRegisteredGateway()
				updatedInitialGateway.Location = "New Foomplandia"
				updatedInitialGateway.MixHost = "100.100.100.100:1789"

				db.RegisterGateway(initialGateway)
				all = db.allRegisteredGateways()
				initRegTime := all[0].RegistrationTime
				assert.Len(GinkgoT(), all, 1)

				db.RegisterGateway(updatedInitialGateway)
				all = db.allRegisteredGateways()

				assert.Len(GinkgoT(), all, 1)
				// since we 'registered' again we should get new registration time
				assert.True(GinkgoT(), all[0].RegistrationTime > initRegTime)

				// this is just so the comparison is easier
				all[0].RegistrationTime = 0
				assert.Equal(GinkgoT(), updatedInitialGateway, all[0])
			})
		})

		Context("Multiple with different identity", func() {
			It("Should not overwrite each other", func() {
				db := NewDb(true)
				all := db.allRegisteredGateways()
				assert.Len(GinkgoT(), all, 0)

				initialGateway1 := fixtures.GoodRegisteredGateway()
				initialGateway2 := fixtures.GoodRegisteredGateway()
				initialGateway2.IdentityKey = "NewID"

				db.RegisterGateway(initialGateway1)
				db.RegisterGateway(initialGateway2)
				all = db.allRegisteredGateways()
				assert.Len(GinkgoT(), all, 2)
			})
		})
	})

	Describe("Removing gateway node", func() {
		Context("If it exists", func() {
			It("Should get rid of it", func() {
				db := NewDb(true)
				all := db.allRegisteredGateways()
				assert.Len(GinkgoT(), all, 0)

				gateway := fixtures.GoodRegisteredGateway()
				db.RegisterGateway(gateway)
				wasRemoved := db.UnregisterNode(gateway.IdentityKey)
				assert.True(GinkgoT(), wasRemoved)

				all = db.allRegisteredGateways()
				assert.Len(GinkgoT(), all, 0)
			})
		})
	})

	Describe("Setting reputation", func() {
		Context("For existing node", func() {
			It("Sets it to defined value", func() {
				db := NewDb(true)
				all := db.allRegisteredMixes()
				assert.Len(GinkgoT(), all, 0)

				mix := fixtures.GoodRegisteredMix()
				db.RegisterMix(mix)
				all = db.allRegisteredMixes()
				assert.Equal(GinkgoT(), all[0].Reputation, int64(0))

				wasChanged := db.SetReputation(mix.IdentityKey, 42)
				assert.True(GinkgoT(), wasChanged)

				all = db.allRegisteredMixes()
				assert.Equal(GinkgoT(), all[0].Reputation, int64(42))
			})
		})

		Context("For non-existent node", func() {
			It("Does nothing", func() {
				db := NewDb(true)
				all := db.allRegisteredMixes()
				assert.Len(GinkgoT(), all, 0)

				wasChanged := db.SetReputation("foomp", 42)
				assert.False(GinkgoT(), wasChanged)
			})
		})
	})

	Describe("Getting topology", func() {
		Context("With no registered nodes", func() {
			It("Returns empty slices", func() {
				db := NewDb(true)
				allMix := db.allRegisteredMixes()
				assert.Len(GinkgoT(), allMix, 0)

				allGate := db.allRegisteredGateways()
				assert.Len(GinkgoT(), allGate, 0)

				topology := db.Topology()
				assert.Len(GinkgoT(), topology.MixNodes, 0)
				assert.Len(GinkgoT(), topology.Gateways, 0)
			})
		})
		Context("With registered nodes", func() {
			It("Returns all registered mixnodes and gateways", func() {
				db := NewDb(true)
				allMix := db.allRegisteredMixes()
				assert.Len(GinkgoT(), allMix, 0)

				allGate := db.allRegisteredGateways()
				assert.Len(GinkgoT(), allGate, 0)

				mix1 := fixtures.GoodRegisteredMix()
				mix2 := fixtures.GoodRegisteredMix()
				mix2.IdentityKey = "aaa"

				gate1 := fixtures.GoodRegisteredGateway()
				gate2 := fixtures.GoodRegisteredGateway()
				gate2.IdentityKey = "bbb"

				db.RegisterMix(mix1)
				db.RegisterMix(mix2)

				db.RegisterGateway(gate1)
				db.RegisterGateway(gate2)

				topology := db.Topology()
				assert.Len(GinkgoT(), topology.MixNodes, 2)
				assert.Len(GinkgoT(), topology.Gateways, 2)
			})
		})
	})

	Describe("Getting active topology", func() {
		Context("With registered nodes but below reputation threshold", func() {
			It("Returns empty slices", func() {
				db := NewDb(true)
				allMix := db.allRegisteredMixes()
				assert.Len(GinkgoT(), allMix, 0)

				allGate := db.allRegisteredGateways()
				assert.Len(GinkgoT(), allGate, 0)

				mix1 := fixtures.GoodRegisteredMix()
				gate1 := fixtures.GoodRegisteredGateway()

				db.RegisterMix(mix1)
				db.RegisterGateway(gate1)

				db.SetReputation(mix1.IdentityKey, ReputationThreshold-1)
				db.SetReputation(gate1.IdentityKey, ReputationThreshold-1)

				topology := db.ActiveTopology(ReputationThreshold)
				assert.Len(GinkgoT(), topology.MixNodes, 0)
				assert.Len(GinkgoT(), topology.Gateways, 0)
			})
		})

		Context("With registered nodes, some above reputation threshold", func() {
			It("Returns only the nodes above the reputation threshold", func() {
				db := NewDb(true)
				allMix := db.allRegisteredMixes()
				assert.Len(GinkgoT(), allMix, 0)

				allGate := db.allRegisteredGateways()
				assert.Len(GinkgoT(), allGate, 0)

				mix1 := fixtures.GoodRegisteredMix()
				mix2 := fixtures.GoodRegisteredMix()
				mix2.IdentityKey = "aaa"

				gate1 := fixtures.GoodRegisteredGateway()
				gate2 := fixtures.GoodRegisteredGateway()
				gate2.IdentityKey = "bbb"

				db.RegisterMix(mix1)
				db.RegisterMix(mix2)

				db.RegisterGateway(gate1)
				db.RegisterGateway(gate2)

				db.SetReputation(mix1.IdentityKey, ReputationThreshold-1)
				db.SetReputation(gate1.IdentityKey, ReputationThreshold-1)
				db.SetReputation(mix2.IdentityKey, ReputationThreshold)
				db.SetReputation(gate2.IdentityKey, ReputationThreshold)

				topology := db.ActiveTopology(ReputationThreshold)
				// this is just so the comparison is easier
				topology.MixNodes[0].RegistrationTime = 0
				topology.Gateways[0].RegistrationTime = 0

				assert.Equal(GinkgoT(), topology.MixNodes[0].Reputation, ReputationThreshold)
				assert.Equal(GinkgoT(), topology.Gateways[0].Reputation, ReputationThreshold)

				topology.MixNodes[0].Reputation = int64(0)
				topology.Gateways[0].Reputation = int64(0)

				assert.Equal(GinkgoT(), topology.MixNodes[0], mix2)
				assert.Equal(GinkgoT(), topology.Gateways[0], gate2)
			})
		})
	})

	Describe("checking for duplicate ips", func() {
		It("works for ipv4", func() {
			ip1 := "1.2.3.4:1789"
			ip2 := "1.2.3.4:1790"
			db := NewDb(true)
			mix1 := fixtures.GoodRegisteredMix()
			mix1.MixHost = ip1

			assert.False(GinkgoT(), db.IpExists(ip1))
			assert.False(GinkgoT(), db.IpExists(ip2))

			db.RegisterMix(mix1)

			assert.True(GinkgoT(), db.IpExists(ip1))
			assert.True(GinkgoT(), db.IpExists(ip2))
		})

		It("works for ipv6", func() {
			ipv6Normal1 := "[2001:0db8:0a0b:12f0:0000:0000:0000:0001]:1789"
			ipv6Normal2 := "[2001:0db8:0a0b:12f0:0000:0000:0000:0001]:1790"
			ipv6Compressed1 := "[2001:db8:a0b:12f0::1]:1789"
			ipv6Compressed2 := "[2001:db8:a0b:12f0::1]:1790"

			db := NewDb(true)
			mix1 := fixtures.GoodRegisteredMix()
			mix1.MixHost = ipv6Normal1

			mix2 := fixtures.GoodRegisteredMix()
			// change id
			mix2.IdentityKey = "foomp"
			mix2.MixHost = ipv6Compressed1

			assert.False(GinkgoT(), db.IpExists(ipv6Normal1))
			assert.False(GinkgoT(), db.IpExists(ipv6Normal2))
			assert.False(GinkgoT(), db.IpExists(ipv6Compressed1))
			assert.False(GinkgoT(), db.IpExists(ipv6Compressed2))

			db.RegisterMix(mix1)
			db.RegisterMix(mix2)

			assert.True(GinkgoT(), db.IpExists(ipv6Normal1))
			assert.True(GinkgoT(), db.IpExists(ipv6Normal2))
			assert.True(GinkgoT(), db.IpExists(ipv6Compressed1))
			assert.True(GinkgoT(), db.IpExists(ipv6Compressed2))
		})

		It("works for domain name", func() {
			name1 := "foomp.com:1789"
			name2 := "foomp.com:1790"

			db := NewDb(true)
			mix1 := fixtures.GoodRegisteredMix()
			mix1.MixHost = name1

			assert.False(GinkgoT(), db.IpExists(name1))
			assert.False(GinkgoT(), db.IpExists(name2))

			db.RegisterMix(mix1)

			assert.True(GinkgoT(), db.IpExists(name1))
			assert.True(GinkgoT(), db.IpExists(name2))
		})

		It("works for gateways", func() {
			ip1 := "1.2.3.4:1789"
			ip2 := "1.2.3.4:1790"

			db := NewDb(true)
			gate1 := fixtures.GoodRegisteredGateway()
			gate1.MixHost = ip1

			assert.False(GinkgoT(), db.IpExists(ip1))
			assert.False(GinkgoT(), db.IpExists(ip2))

			db.RegisterGateway(gate1)

			assert.True(GinkgoT(), db.IpExists(ip1))
			assert.True(GinkgoT(), db.IpExists(ip2))
		})
	})
})
