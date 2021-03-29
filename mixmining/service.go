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
	"sync/atomic"
	"time"

	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/client/rpc"

	"github.com/BorisBorshevsky/timemock"
	"github.com/nymtech/nym/validator/nym/models"
)

// so if you can mix ipv4 but not ipv6, your reputation will go down but not as fast as if you didn't mix at all
const ReportSuccessReputationIncrease = int64(3)
const ReportFailureReputationDecrease = int64(-2)
const ReputationThreshold = int64(100)
const TopologyCacheTTL = time.Second * 30

const Last5MinutesReports = 5
const LastHourReports = 50
const LastDayReports = 1000

const TopologyRefreshing = 1
const TopologyNotRefreshing = 0

// Service struct
type Service struct {
	db     IDb
	cliCtx context.CLIContext

	topology                 models.Topology
	topologyRefreshed        time.Time
	activeTopology           models.Topology
	activeTopologyRefreshed  time.Time
	removedTopology          models.Topology
	removedTopologyRefreshed time.Time

	topologyRefreshing        uint32
	activeTopologyRefreshing  uint32
	removedTopologyRefreshing uint32
}

// IService defines the REST service interface for mixmining.
type IService interface {
	CreateMixStatus(mixStatus models.MixStatus) models.PersistedMixStatus
	ListMixStatus(pubkey string) []models.PersistedMixStatus
	SaveStatusReport(status models.PersistedMixStatus) models.MixStatusReport
	GetStatusReport(pubkey string) models.MixStatusReport

	SaveBatchStatusReport(status []models.PersistedMixStatus) models.BatchMixStatusReport
	BatchCreateMixStatus(batchMixStatus models.BatchMixStatus) []models.PersistedMixStatus
	BatchGetMixStatusReport() models.BatchMixStatusReport

	CheckForDuplicateIP(host string) bool
	MixCount() int
	GatewayCount() int
	GetRemovedTopology() models.Topology
	StartupPurge()
}

// NewService constructor
func NewService(db IDb, isTest bool) *Service {
	service := &Service{
		db:                       db,
		topology:                 db.Topology(),
		topologyRefreshed:        timemock.Now(),
		activeTopology:           db.ActiveTopology(ReputationThreshold),
		activeTopologyRefreshed:  timemock.Now(),
		removedTopology:          db.RemovedTopology(),
		removedTopologyRefreshed: timemock.Now(),
	}

	if !isTest {
		// same with 'last day' report updater (every 10min)
		go lastDayReportsUpdater(service)
		// and old statuses remover (every 1h)
		go oldStatusesPurger(service)
	}

	return service
}

func lastDayReportsUpdater(service *Service) {
	ticker := time.NewTicker(time.Minute * 10)

	for {
		<-ticker.C
		// batchReport := service.updateLastDayReports()
		// service.removeBrokenNodes(&batchReport)
	}

}

func oldStatusesPurger(service *Service) {
	ticker := time.NewTicker(time.Hour * 1)

	for {
		now := timemock.Now()
		lastWeek := now.Add(-(time.Hour * 24 * 7)).UnixNano()
		service.db.RemoveOldStatuses(lastWeek)
		<-ticker.C
	}
}

// func (service *Service) updateLastDayReports() models.BatchMixStatusReport {
// 	dayAgo := timemock.Now().Add(time.Duration(-time.Hour * 24)).UnixNano()
// 	batchReport := service.db.BatchLoadReports(reportKeys)
// 	for idx := range batchReport.Report {
// 		report := &batchReport.Report[idx]
// 		lastDayUptime := service.CalculateUptimeSince(report.PubKey, "4", dayAgo, LastDayReports)
// 		if lastDayUptime == -1 {
// 			// there were no reports to calculate uptime with
// 			continue
// 		}

// 		report.LastDayIPV4 = lastDayUptime
// 		report.LastDayIPV6 = service.CalculateUptimeSince(report.PubKey, "6", dayAgo, LastDayReports)
// 	}

// 	service.db.SaveBatchMixStatusReport(batchReport)
// 	return batchReport
// }

func (service *Service) removeBrokenNodes(batchReport *models.BatchMixStatusReport) {
	// figure out which nodes should get removed
	toRemove := service.batchShouldGetRemoved(batchReport)
	if len(toRemove) > 0 {
		service.db.BatchMoveToRemovedSet(toRemove)
	}
}

// CreateMixStatus adds a new PersistedMixStatus in the orm.
func (service *Service) CreateMixStatus(mixStatus models.MixStatus) models.PersistedMixStatus {
	persistedMixStatus := models.PersistedMixStatus{
		MixStatus: mixStatus,
		Timestamp: timemock.Now().UnixNano(),
	}
	service.db.AddMixStatus(persistedMixStatus)

	return persistedMixStatus
}

// List lists the given number mix metrics
func (service *Service) ListMixStatus(pubkey string) []models.PersistedMixStatus {
	return service.db.ListMixStatus(pubkey, 1000)
}

// GetStatusReport gets a single MixStatusReport by node public key
func (service *Service) GetStatusReport(pubkey string) models.MixStatusReport {
	return service.db.LoadReport(pubkey)
}

// BatchCreateMixStatus batch adds new multiple PersistedMixStatus in the orm.
func (service *Service) BatchCreateMixStatus(batchMixStatus models.BatchMixStatus) []models.PersistedMixStatus {
	statusList := make([]models.PersistedMixStatus, len(batchMixStatus.Status))
	for i, mixStatus := range batchMixStatus.Status {
		persistedMixStatus := models.PersistedMixStatus{
			MixStatus: mixStatus,
			Timestamp: timemock.Now().UnixNano(),
		}
		statusList[i] = persistedMixStatus
	}
	service.db.BatchAddMixStatus(statusList)

	return statusList
}

// BatchGetMixStatusReport gets BatchMixStatusReport which contain multiple MixStatusReport.
func (service *Service) BatchGetMixStatusReport() models.BatchMixStatusReport {
	return service.db.LoadNonStaleReports()
}

// SaveBatchStatusReport builds and saves a status report for multiple mixnodes simultaneously.
// Those reports can be updated once whenever we receive a new status,
// and the saved results can then be queried. This keeps us from having to build the report dynamically
// on every request at runtime.
func (service *Service) SaveBatchStatusReport(status []models.PersistedMixStatus) models.BatchMixStatusReport {
	pubkeys := make([]string, len(status))
	for i := range status {
		pubkeys[i] = status[i].PubKey
	}
	batchReport := service.db.BatchLoadReports(pubkeys)

	// that's super crude but I don't think db results are guaranteed to come in order, plus some entries might
	// not exist
	reportMap := make(map[string]int)
	reputationChangeMap := make(map[string]int64)
	for i, report := range batchReport.Report {
		reportMap[report.PubKey] = i
	}

	for _, mixStatus := range status {
		if reportIdx, ok := reportMap[mixStatus.PubKey]; ok {
			service.updateReportUpToLastHour(&batchReport.Report[reportIdx], &mixStatus)
			if *mixStatus.Up {
				reputationChangeMap[mixStatus.PubKey] += ReportSuccessReputationIncrease
			} else {
				reputationChangeMap[mixStatus.PubKey] += ReportFailureReputationDecrease
			}
		} else {
			var freshReport models.MixStatusReport
			service.updateReportUpToLastHour(&freshReport, &mixStatus)
			batchReport.Report = append(batchReport.Report, freshReport)
			reportMap[freshReport.PubKey] = len(batchReport.Report) - 1
			if *mixStatus.Up {
				reputationChangeMap[mixStatus.PubKey] = ReportSuccessReputationIncrease
			} else {
				reputationChangeMap[mixStatus.PubKey] = ReportFailureReputationDecrease
			}
		}
	}

	service.db.SaveBatchMixStatusReport(batchReport)
	service.db.BatchUpdateReputation(reputationChangeMap)

	return batchReport
}

func (service *Service) updateReportUpToLastHour(report *models.MixStatusReport, status *models.PersistedMixStatus) {
	report.PubKey = status.PubKey // crude, we do this in case it's a fresh struct returned from the db

	if status.IPVersion == "4" {
		report.MostRecentIPV4 = *status.Up
		report.Last5MinutesIPV4 = service.CalculateUptime(status.PubKey, "4", Last5MinutesReports)
		report.LastHourIPV4 = service.CalculateUptime(status.PubKey, "4", LastHourReports)
	} else if status.IPVersion == "6" {
		report.MostRecentIPV6 = *status.Up
		report.Last5MinutesIPV6 = service.CalculateUptime(status.PubKey, "6", Last5MinutesReports)
		report.LastHourIPV6 = service.CalculateUptime(status.PubKey, "6", LastHourReports)
	}
}

// SaveStatusReport builds and saves a status report for a mixnode. The report can be updated once
// whenever we receive a new status, and the saved result can then be queried. This keeps us from
// having to build the report dynamically on every request at runtime.
func (service *Service) SaveStatusReport(status models.PersistedMixStatus) models.MixStatusReport {
	report := service.db.LoadReport(status.PubKey)

	service.updateReportUpToLastHour(&report, &status)
	service.db.SaveMixStatusReport(report)

	if *status.Up {
		service.db.UpdateReputation(status.PubKey, ReportSuccessReputationIncrease)
		// if the status was up, there's no way the quality has decreased
	} else {
		service.db.UpdateReputation(status.PubKey, ReportFailureReputationDecrease)
		if service.shouldGetRemoved(&report) {
			service.db.MoveToRemovedSet(report.PubKey)
		}
	}

	return report
}

// shouldGetRemoved is called upon receiving mix status for this particular node. It determines whether the node is still
// eligible to be part of the main topology or should moved into 'removed set'
func (service *Service) shouldGetRemoved(report *models.MixStatusReport) bool {
	// check if last 24h ipv4 uptime is > 50%
	if report.LastDayIPV4 < 50 {
		return true
	}

	// if it ever mixed any ipv6 packet, do the same check for ipv6 uptime
	if report.LastDayIPV6 > 0 && report.LastDayIPV6 < 50 {
		return true
	}

	// TODO: does it make sense to also check reputation here? But if we do it, then each new node would get
	// removed immediately before they even get a chance to build it up

	return false
}

// batchShouldGetRemoved is called upon receiving batch mix status for the set of those particular nodes.
// It determines whether the nodes are still eligible to be part of the main topology or should moved into 'removed set'
func (service *Service) batchShouldGetRemoved(batchReport *models.BatchMixStatusReport) []string {
	broken := make([]string, 0)

	for _, report := range batchReport.Report {
		// check if last 24h ipv4 uptime is > 50%
		if report.LastDayIPV4 < 50 {
			broken = append(broken, report.PubKey)
			continue
		}

		// if it ever mixed any ipv6 packet, do the same check for ipv6 uptime
		if report.LastDayIPV6 > 0 && report.LastDayIPV6 < 50 {
			broken = append(broken, report.PubKey)
			continue
		}

		// TODO: does it make sense to also check reputation here? But if we do it, then each new node would get
		// removed immediately before they even get a chance to build it up
	}

	return broken
}

// CalculateUptime calculates percentage uptime for a given node, protocol since a specific time
func (service *Service) CalculateUptime(pubkey string, ipVersion string, numReports int) int {
	statuses := service.db.GetNMostRecentMixStatuses(pubkey, ipVersion, numReports)
	numStatuses := len(statuses)
	if numStatuses == 0 {
		// this can only happen to the goroutine calculating uptime for last 1000 reports
		return -1
	}
	up := 0
	for _, status := range statuses {
		if *status.Up {
			up = up + 1
		}
	}

	return service.calculatePercent(up, numStatuses)
}

func (service *Service) CalculateUptimeSince(pubkey string, ipVersion string, since int64, numReports int) int {
	statuses := service.db.ListMixStatusSinceWithLimit(pubkey, ipVersion, since, numReports)
	numStatuses := len(statuses)
	if numStatuses == 0 {
		// this can only happen to the goroutine calculating uptime for last 1000 reports
		return -1
	}
	up := 0
	for _, status := range statuses {
		if *status.Up {
			up = up + 1
		}
	}

	return service.calculatePercent(up, numStatuses)
}

func (service *Service) calculatePercent(num int, outOf int) int {
	return int(float32(num) / float32(outOf) * 100)
}

func (service *Service) CheckForDuplicateIP(host string) bool {
	return service.db.IpExists(host)
}

func emptyValidators() rpc.ResultValidatorsOutput {
	return rpc.ResultValidatorsOutput{
		BlockHeight: 0,
		Validators:  []rpc.ValidatorOutput{},
	}
}

func (service *Service) MixCount() int {
	topology := service.db.Topology()
	return len(topology.MixNodes)
}

func (service *Service) GatewayCount() int {
	topology := service.db.Topology()
	return len(topology.Gateways)
}

func (service *Service) GetRemovedTopology() models.Topology {
	now := timemock.Now()
	if now.Sub(service.removedTopologyRefreshed) > TopologyCacheTTL {
		// if topology is not refreshing, start refreshing
		if atomic.CompareAndSwapUint32(&service.removedTopologyRefreshing, TopologyNotRefreshing, TopologyRefreshing) {
			// put in defer block to ensure it's going to get called if something crashes
			defer func() {
				service.removedTopologyRefreshing = TopologyNotRefreshing
			}()

			newTopology := service.db.RemovedTopology()
			service.removedTopology = newTopology
			service.removedTopologyRefreshed = now
		}
	}

	return service.removedTopology
}

// StartupPurge moves any mixnode from the main topology into 'removed' if it is not running
// version 0.9.2. The "50%" uptime requirement does not need to be checked here as if it's
// not fulfilled, the node will be automatically moved to "removed set" on the first
// run of the network monitor
func (service *Service) StartupPurge() {
	nodesToRemove := make([]string, 0)
	topology := service.db.Topology()
	for _, mix := range topology.MixNodes {
		if mix.Version != SystemVersion {
			nodesToRemove = append(nodesToRemove, mix.IdentityKey)
		}
	}
	for _, gateway := range topology.Gateways {
		if gateway.Version != SystemVersion {
			nodesToRemove = append(nodesToRemove, gateway.IdentityKey)
		}
	}
	service.db.BatchMoveToRemovedSet(nodesToRemove)
}
