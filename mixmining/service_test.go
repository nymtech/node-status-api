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
	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/nymtech/nym/validator/nym/directory/mixmining/fixtures"
	"github.com/nymtech/nym/validator/nym/directory/mixmining/mocks"
	"github.com/nymtech/nym/validator/nym/directory/models"
	. "github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
	"net/http"
	"time"
)

func now() int64 {
	return timemock.Now().UnixNano()
}

func daysAgo(days int) int64 {
	now := timemock.Now()
	return now.Add(time.Duration(-days) * time.Hour * 24).UnixNano()
}

func minutesAgo(minutes int) int64 {
	now := timemock.Now()
	return now.Add(time.Duration(-minutes) * time.Minute).UnixNano()
}

// Some fixtures data to dry up tests a bit

// A slice of IPv4 mix statuses with 2 ups and 1 down during the past day
func twoUpOneDown() []models.PersistedMixStatus {
	db := []models.PersistedMixStatus{}
	var status = persistedStatus()

	booltrue := true
	status.PubKey = "key1"
	status.IPVersion = "4"
	status.Up = &booltrue

	status.Timestamp = minutesAgo(5)
	db = append(db, status)

	status.Timestamp = minutesAgo(10)
	db = append(db, status)

	boolfalse := false
	status.Timestamp = minutesAgo(15)
	status.Up = &boolfalse
	db = append(db, status)

	return db
}

func persistedStatus() models.PersistedMixStatus {
	mixStatus := status()
	persisted := models.PersistedMixStatus{
		MixStatus: mixStatus,
		Timestamp: Now(),
	}
	return persisted
}

func persistedStatusDown(key string, ipversion string) models.PersistedMixStatus {
	mixStatus := statusDown(key, ipversion)
	persisted := models.PersistedMixStatus{
		MixStatus: mixStatus,
		Timestamp: Now(),
	}
	return persisted
}

func status() models.MixStatus {
	boolfalse := false
	return models.MixStatus{
		PubKey:    "key1",
		IPVersion: "4",
		Up:        &boolfalse,
	}
}

func statusUp(key string, ipversion string) models.MixStatus {
	booltrue := true
	return models.MixStatus{
		PubKey:    key,
		IPVersion: ipversion,
		Up:        &booltrue,
	}
}

func statusDown(key string, ipversion string) models.MixStatus {
	boolfalse := false
	return models.MixStatus{
		PubKey:    key,
		IPVersion: ipversion,
		Up:        &boolfalse,
	}
}

func persistedStatusFrom(mixStatus models.MixStatus) models.PersistedMixStatus {
	persisted := models.PersistedMixStatus{
		MixStatus: mixStatus,
		Timestamp: Now(),
	}
	return persisted
}

// A version of now with a frozen shared clock so we can have determinate time-based tests
func Now() int64 {
	now := timemock.Now()
	timemock.Freeze(now) //time is frozen
	nanos := now.UnixNano()
	return nanos
}

var _ = Describe("mixmining.Service", func() {
	var mockDb mocks.IDb
	var status1 models.MixStatus
	var status2 models.MixStatus
	var persisted1 models.PersistedMixStatus
	var persisted2 models.PersistedMixStatus

	var serv Service

	boolfalse := false
	booltrue := true

	status1 = models.MixStatus{
		PubKey:    "key1",
		IPVersion: "4",
		Up:        &boolfalse,
	}

	persisted1 = models.PersistedMixStatus{
		MixStatus: status1,
		Timestamp: Now(),
	}

	status2 = models.MixStatus{
		PubKey:    "key2",
		IPVersion: "6",
		Up:        &booltrue,
	}

	persisted2 = models.PersistedMixStatus{
		MixStatus: status2,
		Timestamp: Now(),
	}

	downer := persisted1
	downer.MixStatus.Up = &boolfalse

	upper := persisted1
	upper.MixStatus.Up = &booltrue

	persistedList := []models.PersistedMixStatus{persisted1, persisted2}
	emptyList := []models.PersistedMixStatus{}

	BeforeEach(func() {
		mockDb = *new(mocks.IDb)
		mockDb.On("Topology").Return(models.Topology{})
		mockDb.On("ActiveTopology", ReputationThreshold).Return(models.Topology{}).Once()
		mockDb.On("RemovedTopology").Return(models.Topology{})
		serv = *NewService(&mockDb, context.NewCLIContext(), true)
	})

	Describe("Adding a mix status and creating a new summary report for a node", func() {
		Context("when no statuses have yet been saved", func() {
			It("should add a PersistedMixStatus to the db and save the new report", func() {

				mockDb.On("AddMixStatus", persisted1)

				serv.CreateMixStatus(status1)
				mockDb.AssertCalled(GinkgoT(), "AddMixStatus", persisted1)
			})
		})
	})
	Describe("Listing mix statuses", func() {
		Context("when receiving a list request", func() {
			It("should call to the Db", func() {
				mockDb.On("ListMixStatus", persisted1.PubKey, 1000).Return(persistedList)

				result := serv.ListMixStatus(persisted1.PubKey)

				mockDb.AssertCalled(GinkgoT(), "ListMixStatus", persisted1.PubKey, 1000)
				assert.Equal(GinkgoT(), persistedList[0].MixStatus.PubKey, result[0].MixStatus.PubKey)
				assert.Equal(GinkgoT(), persistedList[1].MixStatus.PubKey, result[1].MixStatus.PubKey)
			})
		})
	})

	Describe("Calculating uptime", func() {
		Context("when no statuses exist yet", func() {
			It("should return 0", func() {
				mockDb.On("ListMixStatusSinceWithLimit", "key1", "4", daysAgo(30), LastDayReports*30).Return(emptyList)

				uptime := serv.CalculateUptimeSince(persisted1.PubKey, persisted1.IPVersion, daysAgo(30), LastDayReports*30)
				assert.Equal(GinkgoT(), -1, uptime)
			})

		})
		Context("when 2 ups and 1 down exist in the given time period", func() {
			It("should return 66", func() {
				mockDb.On("ListMixStatusSinceWithLimit", "key1", "4", daysAgo(1), LastDayReports).Return(twoUpOneDown())

				uptime := serv.CalculateUptimeSince("key1", "4", daysAgo(1), LastDayReports)
				expected := 66 // percent
				assert.Equal(GinkgoT(), expected, uptime)
			})
		})
	})

	Describe("Saving a mix status report", func() {
		Context("when 1 down status exists", func() {
			BeforeEach(func() {
				oneDown := []models.PersistedMixStatus{downer}
				mockDb.On("GetNMostRecentMixStatuses", downer.PubKey, downer.IPVersion, Last5MinutesReports).Return(oneDown)
				mockDb.On("GetNMostRecentMixStatuses", downer.PubKey, downer.IPVersion, LastHourReports).Return(oneDown)
			})
			Context("this one *must be* a downer, so calculate using it", func() {
				BeforeEach(func() {
					mockDb.On("LoadReport", downer.PubKey).Return(models.MixStatusReport{}) // TODO: Mockery isn't happy returning an untyped nil, so I've had to sub in a blank `models.MixStatusReport{}`. It will actually return a nil.
					expectedSave := models.MixStatusReport{
						PubKey:           downer.PubKey,
						MostRecentIPV4:   false,
						Last5MinutesIPV4: 0,
						LastHourIPV4:     0,
						LastDayIPV4:      0,
						MostRecentIPV6:   false,
						Last5MinutesIPV6: 0,
						LastHourIPV6:     0,
						LastDayIPV6:      0,
					}
					mockDb.On("UpdateReputation", downer.PubKey, ReportFailureReputationDecrease).Return(true)
					mockDb.On("SaveMixStatusReport", expectedSave)
				})
				It("should save the initial report, all statuses will be set to down. Node will also be moved to removed set", func() {
					mockDb.On("MoveToRemovedSet", downer.PubKey)
					result := serv.SaveStatusReport(downer)
					assert.Equal(GinkgoT(), 0, result.Last5MinutesIPV4)
					assert.Equal(GinkgoT(), 0, result.LastHourIPV4)
					assert.Equal(GinkgoT(), 0, result.LastDayIPV4)
					mockDb.AssertExpectations(GinkgoT())
				})
			})

		})
		Context("when 1 up status exists", func() {
			BeforeEach(func() {
				oneUp := []models.PersistedMixStatus{upper}
				mockDb.On("GetNMostRecentMixStatuses", downer.PubKey, downer.IPVersion, Last5MinutesReports).Return(oneUp)
				mockDb.On("GetNMostRecentMixStatuses", downer.PubKey, downer.IPVersion, LastHourReports).Return(oneUp)
			})
			Context("this one *must be* an upper, so calculate using it", func() {
				BeforeEach(func() {
					oneDown := []models.PersistedMixStatus{downer}
					mockDb.On("GetNMostRecentMixStatuses", upper.PubKey, upper.IPVersion, Last5MinutesReports, now()).Return(oneDown)
					mockDb.On("GetNMostRecentMixStatuses", upper.PubKey, upper.IPVersion, LastHourReports, now()).Return(oneDown)
					mockDb.On("LoadReport", upper.PubKey).Return(models.MixStatusReport{}) // TODO: Mockery isn't happy returning an untyped nil, so I've had to sub in a blank `models.MixStatusReport{}`. It will actually return a nil.
					expectedSave := models.MixStatusReport{
						PubKey:           upper.PubKey,
						MostRecentIPV4:   true,
						Last5MinutesIPV4: 100,
						LastHourIPV4:     100,
						LastDayIPV4:      0,
						MostRecentIPV6:   false,
						Last5MinutesIPV6: 0,
						LastHourIPV6:     0,
						LastDayIPV6:      0,
					}
					mockDb.On("UpdateReputation", upper.PubKey, ReportSuccessReputationIncrease).Return(true)
					mockDb.On("SaveMixStatusReport", expectedSave)
				})
				It("should save the initial report, all statuses will be set to up", func() {
					result := serv.SaveStatusReport(upper)
					assert.Equal(GinkgoT(), true, result.MostRecentIPV4)
					assert.Equal(GinkgoT(), 100, result.Last5MinutesIPV4)
					assert.Equal(GinkgoT(), 100, result.LastHourIPV4)
					assert.Equal(GinkgoT(), 0, result.LastDayIPV4)
				})
			})
		})

		Context("when 2 up statuses exist for the last 5 minutes already and we just added a down", func() {
			BeforeEach(func() {
				mockDb.On("GetNMostRecentMixStatuses", downer.PubKey, downer.IPVersion, Last5MinutesReports).Return(twoUpOneDown())
				mockDb.On("GetNMostRecentMixStatuses", downer.PubKey, downer.IPVersion, LastHourReports).Return(twoUpOneDown())
			})
			It("should save the report", func() {
				initialState := models.MixStatusReport{
					PubKey:           downer.PubKey,
					MostRecentIPV4:   true,
					Last5MinutesIPV4: 100,
					LastHourIPV4:     100,
					LastDayIPV4:      100,
					MostRecentIPV6:   false,
					Last5MinutesIPV6: 0,
					LastHourIPV6:     0,
					LastDayIPV6:      0,
				}

				expectedAfterUpdate := models.MixStatusReport{
					PubKey:           downer.PubKey,
					MostRecentIPV4:   false,
					Last5MinutesIPV4: 66,
					LastHourIPV4:     66,
					LastDayIPV4:      100, // last day will not change, it's updated in separate routine
					MostRecentIPV6:   false,
					Last5MinutesIPV6: 0,
					LastHourIPV6:     0,
					LastDayIPV6:      0,
				}
				mockDb.On("LoadReport", downer.PubKey).Return(initialState)
				mockDb.On("SaveMixStatusReport", expectedAfterUpdate)
				mockDb.On("UpdateReputation", downer.PubKey, ReportFailureReputationDecrease).Return(true)

				updatedStatus := serv.SaveStatusReport(downer)
				assert.Equal(GinkgoT(), expectedAfterUpdate, updatedStatus)
				mockDb.AssertCalled(GinkgoT(), "UpdateReputation", downer.PubKey, ReportFailureReputationDecrease)

				mockDb.AssertExpectations(GinkgoT())
			})
		})
	})

	Describe("Saving batch status report", func() {
		Context("if it contains v4 and v6 up status for same node", func() {
			It("should combine them into single entry. Node will also be moved to removed set", func() {
				upv4 := persistedStatusFrom(statusDown("key1", "4"))
				upv6 := persistedStatusFrom(statusDown("key1", "6"))
				batchReport := []models.PersistedMixStatus{upv4, upv6}

				expected := models.BatchMixStatusReport{
					Report: []models.MixStatusReport{{
						PubKey:           "key1",
						MostRecentIPV4:   false,
						Last5MinutesIPV4: 0,
						LastHourIPV4:     0,
						LastDayIPV4:      0,
						MostRecentIPV6:   false,
						Last5MinutesIPV6: 0,
						LastHourIPV6:     0,
						LastDayIPV6:      0,
					}},
				}

				mockDb.On("GetNMostRecentMixStatuses", "key1", "4", Last5MinutesReports).Return([]models.PersistedMixStatus{persistedStatusDown("key1", "4")})
				mockDb.On("GetNMostRecentMixStatuses", "key1", "4", LastHourReports).Return([]models.PersistedMixStatus{persistedStatusDown("key1", "4")})
				mockDb.On("GetNMostRecentMixStatuses", "key1", "6", Last5MinutesReports).Return([]models.PersistedMixStatus{persistedStatusDown("key1", "6")})
				mockDb.On("GetNMostRecentMixStatuses", "key1", "6", LastHourReports).Return([]models.PersistedMixStatus{persistedStatusDown("key1", "6")})

				mockDb.On("BatchLoadReports", []string{"key1", "key1"}).Return(models.BatchMixStatusReport{Report: make([]models.MixStatusReport, 0)})
				mockDb.On("SaveBatchMixStatusReport", expected)
				mockDb.On("BatchUpdateReputation", map[string]int64{"key1": 2 * ReportFailureReputationDecrease})
				mockDb.On("BatchMoveToRemovedSet", []string{"key1"})
				updatedStatus := serv.SaveBatchStatusReport(batchReport)
				assert.Equal(GinkgoT(), 1, len(updatedStatus.Report))
				mockDb.AssertCalled(GinkgoT(), "BatchUpdateReputation", map[string]int64{"key1": 2 * ReportFailureReputationDecrease})
			})
		})
	})

	Describe("Getting a mix status report", func() {
		Context("When no saved report exists for a pubkey", func() {
			It("should return an empty report", func() {
				blank := models.MixStatusReport{}
				mockDb.On("LoadReport", "superkey").Return(blank)

				report := serv.GetStatusReport("superkey")
				assert.Equal(GinkgoT(), blank, report)
			})
		})
		Context("When a saved report exists for a pubkey", func() {
			It("should return the report", func() {
				perfect := models.MixStatusReport{
					PubKey:           "superkey",
					MostRecentIPV4:   true,
					Last5MinutesIPV4: 100,
					LastHourIPV4:     100,
					LastDayIPV4:      100,
					MostRecentIPV6:   true,
					Last5MinutesIPV6: 100,
					LastHourIPV6:     100,
					LastDayIPV6:      100,
				}
				mockDb.On("LoadReport", "superkey").Return(perfect)

				report := serv.GetStatusReport("superkey")
				assert.Equal(GinkgoT(), perfect, report)
			})
		})
	})
})

var _ = Describe("mixmining.registration.Service", func() {
	var mockDb *mocks.IDb
	var serv *Service

	BeforeEach(func() {
		mockDb = &mocks.IDb{}
		mockDb.On("Topology").Return(models.Topology{})
		mockDb.On("ActiveTopology", ReputationThreshold).Return(models.Topology{})
		mockDb.On("RemovedTopology").Return(models.Topology{})
		serv = NewService(mockDb, context.NewCLIContext(), true)
	})

	Describe("Adding mix registration info", func() {
		It("creates new registered mix with empty reputation and zero timestamp", func() {
			info := fixtures.GoodMixRegistrationInfo()
			registeredMix := models.RegisteredMix{
				MixRegistrationInfo: info,
			}

			mockDb.On("RegisterMix", registeredMix)
			serv.RegisterMix(info)
			mockDb.AssertCalled(GinkgoT(), "RegisterMix", registeredMix)
		})
	})

	Describe("Adding gateway registration info", func() {
		It("creates new registered gateway with empty reputation and zero timestamp", func() {
			info := fixtures.GoodGatewayRegistrationInfo()
			registeredGateway := models.RegisteredGateway{
				GatewayRegistrationInfo: info,
			}

			mockDb.On("RegisterGateway", registeredGateway)
			serv.RegisterGateway(info)
			mockDb.AssertCalled(GinkgoT(), "RegisterGateway", registeredGateway)
		})
	})

	Describe("Unregistering node", func() {
		Context("When caller ip matches", func () {
			It("Performs unregistration for node announcing ip address", func () {
				nodeID := "foomp"

				mockDb.On("GetNodeMixHost", nodeID).Return("127.0.0.1:1234")
				mockDb.On("UnregisterNode", nodeID).Return(true)

				_, err := serv.UnregisterNode(nodeID, "127.0.0.1")
				assert.Nil(GinkgoT(), err)
				mockDb.AssertCalled(GinkgoT(), "UnregisterNode", nodeID)
			})

			It("Performs unregistration for node announcing hostname", func () {
				nodeID := "foomp"

				mockDb.On("GetNodeMixHost", nodeID).Return("nymtech.net:1234")
				mockDb.On("UnregisterNode", nodeID).Return(true)

				_, err := serv.UnregisterNode(nodeID, "185.19.28.43")
				assert.Nil(GinkgoT(), err)
				mockDb.AssertCalled(GinkgoT(), "UnregisterNode", nodeID)
			})
		})

		Context("When caller ip is empty", func() {
			It("returns a bad status", func() {
				nodeID := "foomp"

				status, err := serv.UnregisterNode(nodeID, "")
				assert.NotNil(GinkgoT(), status, http.StatusBadRequest)
				assert.NotNil(GinkgoT(), err)
			})
		})

		Context("When node with specified id doesn't exist", func() {
			It("returns a not found status", func() {
				nodeID := "foomp"

				mockDb.On("GetNodeMixHost", nodeID).Return("")

				status, err := serv.UnregisterNode(nodeID, "1.1.1.1")
				assert.NotNil(GinkgoT(), status, http.StatusBadRequest)
				assert.NotNil(GinkgoT(), err)
			})
		})

		Context("When caller ip doesn't match", func () {
			It("Doesn't perform unregistration for node announcing ip address", func () {
				nodeID := "foomp"

				mockDb.On("GetNodeMixHost", nodeID).Return("127.0.0.1:1234")

				status, err := serv.UnregisterNode(nodeID, "1.1.1.1")
				assert.NotNil(GinkgoT(), status, http.StatusForbidden)
				assert.NotNil(GinkgoT(), err)
			})

			It("Doesn't perform unregistration for node announcing hostname", func () {
				nodeID := "foomp"

				mockDb.On("GetNodeMixHost", nodeID).Return("nymtech.net:1234")

				status, err := serv.UnregisterNode(nodeID, "1.1.1.1")
				assert.NotNil(GinkgoT(), status, http.StatusForbidden)
				assert.NotNil(GinkgoT(), err)
			})
		})
	})

	Describe("Setting reputation of a node", func() {
		Context("With given identity when it exists", func() {
			It("Calls internal database with correct arguments", func() {
				nodeID := "foomp"
				newRep := int64(42)
				mockDb.On("SetReputation", nodeID, newRep).Return(true)

				assert.True(GinkgoT(), serv.SetReputation(nodeID, newRep))
				mockDb.AssertCalled(GinkgoT(), "SetReputation", nodeID, newRep)
			})
		})

		Context("With given identity when it doesn't exists", func() {
			It("Calls internal database with correct arguments", func() {
				nodeID := "foomp"
				newRep := int64(42)
				mockDb.On("SetReputation", nodeID, newRep).Return(false)

				assert.False(GinkgoT(), serv.SetReputation(nodeID, newRep))
				mockDb.AssertCalled(GinkgoT(), "SetReputation", nodeID, newRep)
			})
		})
	})
})
