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
	"time"

	"github.com/BorisBorshevsky/timemock"
	"github.com/nymtech/node-status-api/models"
)

// Service struct
type Service struct {
	db     IDb
}

// IService defines the REST service interface for mixmining.
type IService interface {
	CreateMixStatus(mixStatus models.MixStatus) models.PersistedMixStatus
	ListMixStatus(pubkey string) []models.PersistedMixStatus
	SaveMixStatusReport(status models.PersistedMixStatus) models.MixStatusReport
	GetMixStatusReport(pubkey string) models.MixStatusReport

	SaveBatchMixStatusReport(status []models.PersistedMixStatus) models.BatchMixStatusReport
	BatchCreateMixStatus(batchMixStatus models.BatchMixStatus) []models.PersistedMixStatus
	BatchGetMixStatusReport() models.BatchMixStatusReport


	CreateGatewayStatus(gatewayStatus models.GatewayStatus) models.PersistedGatewayStatus
	ListGatewayStatus(pubkey string) []models.PersistedGatewayStatus
	SaveGatewayStatusReport(status models.PersistedGatewayStatus) models.GatewayStatusReport
	GetGatewayStatusReport(pubkey string) models.GatewayStatusReport

	SaveBatchGatewayStatusReport(status []models.PersistedGatewayStatus) models.BatchGatewayStatusReport
	BatchCreateGatewayStatus(batchGatewayStatus models.BatchGatewayStatus) []models.PersistedGatewayStatus
	BatchGetGatewayStatusReport() models.BatchGatewayStatusReport
}

// NewService constructor
func NewService(db IDb, isTest bool) *Service {
	service := &Service{
		db: db,
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
		fmt.Println("Updating last day reports")
		service.updateLastDayMixReports()
		service.updateLastDayGatewayReports()
	}

}

func oldStatusesPurger(service *Service) {
	ticker := time.NewTicker(time.Hour * 1)

	for {
		now := timemock.Now()
		lastWeek := now.Add(-(time.Hour * 24 * 7)).UnixNano()
		service.db.RemoveOldMixStatuses(lastWeek)
		service.db.RemoveOldGatewayStatuses(lastWeek)
		<-ticker.C
	}
}

func (service *Service) updateLastDayMixReports() models.BatchMixStatusReport {
	dayAgo := timemock.Now().Add(-time.Hour * 24).UnixNano()
	allActive := service.db.GetActiveMixes(dayAgo)

	batchReport := service.db.BatchLoadMixReports(allActive)

	for i := range batchReport.Report {
		lastDayUptime := service.CalculateMixUptime(batchReport.Report[i].PubKey, "4", dayAgo)
		if lastDayUptime == -1 {
			// there were no reports to calculate uptime with
			// but this should NEVER happen as we only loaded reports for the nodes that received status data
			// in last 24h
			continue
		}

		batchReport.Report[i].LastDayIPV4 = lastDayUptime
		batchReport.Report[i].LastDayIPV6 = service.CalculateMixUptime(batchReport.Report[i].PubKey, "6", dayAgo)
	}

	service.db.SaveBatchMixStatusReport(batchReport)
	return batchReport
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
func (service *Service) GetMixStatusReport(pubkey string) models.MixStatusReport {
	return service.db.LoadMixReport(pubkey)
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
	return service.db.LoadNonStaleMixReports()
}

// SaveBatchStatusReport builds and saves a status report for multiple mixnodes simultaneously.
// Those reports can be updated once whenever we receive a new status,
// and the saved results can then be queried. This keeps us from having to build the report dynamically
// on every request at runtime.
func (service *Service) SaveBatchMixStatusReport(status []models.PersistedMixStatus) models.BatchMixStatusReport {
	pubkeys := make([]string, len(status))
	for i := range status {
		pubkeys[i] = status[i].PubKey
	}
	batchReport := service.db.BatchLoadMixReports(pubkeys)

	// that's super crude but I don't think db results are guaranteed to come in order, plus some entries might
	// not exist
	reportMap := make(map[string]int)
	for i, report := range batchReport.Report {
		reportMap[report.PubKey] = i
	}

	for _, mixStatus := range status {
		if reportIdx, ok := reportMap[mixStatus.PubKey]; ok {
			service.updateMixReportUpToLastHour(&batchReport.Report[reportIdx], &mixStatus)
		} else {
			var freshReport models.MixStatusReport
			service.updateMixReportUpToLastHour(&freshReport, &mixStatus)
			batchReport.Report = append(batchReport.Report, freshReport)
			reportMap[freshReport.PubKey] = len(batchReport.Report) - 1
		}
	}

	service.db.SaveBatchMixStatusReport(batchReport)

	return batchReport
}

// SaveStatusReport builds and saves a status report for a mixnode. The report can be updated once
// whenever we receive a new status, and the saved result can then be queried. This keeps us from
// having to build the report dynamically on every request at runtime.
func (service *Service) SaveMixStatusReport(status models.PersistedMixStatus) models.MixStatusReport {
	report := service.db.LoadMixReport(status.PubKey)

	service.updateMixReportUpToLastHour(&report, &status)
	service.db.SaveMixStatusReport(report)

	return report
}

func (service *Service) updateMixReportUpToLastHour(report *models.MixStatusReport, status *models.PersistedMixStatus) {
	report.PubKey = status.PubKey // crude, we do this in case it's a fresh struct returned from the db
	report.Owner = status.Owner

	if status.IPVersion == "4" {
		report.MostRecentIPV4 = *status.Up
		report.Last5MinutesIPV4 = service.CalculateMixUptime(status.PubKey, "4", minutesAgo(5))
		report.LastHourIPV4 = service.CalculateMixUptime(status.PubKey, "4", minutesAgo(60))
	} else if status.IPVersion == "6" {
		report.MostRecentIPV6 = *status.Up
		report.Last5MinutesIPV6 = service.CalculateMixUptime(status.PubKey, "6", minutesAgo(5))
		report.LastHourIPV6 = service.CalculateMixUptime(status.PubKey, "6", minutesAgo(60))
	}
}

func (service *Service) CalculateMixUptime(pubkey string, ipVersion string, since int64) int {
	statuses := service.db.ListMixStatusSince(pubkey, ipVersion, since)
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


func (service *Service) updateLastDayGatewayReports() models.BatchGatewayStatusReport {
	dayAgo := timemock.Now().Add(-time.Hour * 24).UnixNano()
	allActive := service.db.GetActiveGateways(dayAgo)

	batchReport := service.db.BatchLoadGatewayReports(allActive)

	for i := range batchReport.Report {
		lastDayUptime := service.CalculateGatewayUptime(batchReport.Report[i].PubKey, "4", dayAgo)
		if lastDayUptime == -1 {
			// there were no reports to calculate uptime with
			// but this should NEVER happen as we only loaded reports for the nodes that received status data
			// in last 24h
			continue
		}

		batchReport.Report[i].LastDayIPV4 = lastDayUptime
		batchReport.Report[i].LastDayIPV6 = service.CalculateGatewayUptime(batchReport.Report[i].PubKey, "6", dayAgo)
	}

	service.db.SaveBatchGatewayStatusReport(batchReport)
	return batchReport
}

// CreateGatewayStatus adds a new PersistedGatewayStatus in the orm.
func (service *Service) CreateGatewayStatus(gatewayStatus models.GatewayStatus) models.PersistedGatewayStatus {
	persistedGatewayStatus := models.PersistedGatewayStatus{
		GatewayStatus: gatewayStatus,
		Timestamp: timemock.Now().UnixNano(),
	}
	service.db.AddGatewayStatus(persistedGatewayStatus)

	return persistedGatewayStatus
}

// List lists the given number gateway metrics
func (service *Service) ListGatewayStatus(pubkey string) []models.PersistedGatewayStatus {
	return service.db.ListGatewayStatus(pubkey, 1000)
}

// GetStatusReport gets a single GatewayStatusReport by node public key
func (service *Service) GetGatewayStatusReport(pubkey string) models.GatewayStatusReport {
	return service.db.LoadGatewayReport(pubkey)
}

// BatchCreateGatewayStatus batch adds new multiple PersistedGatewayStatus in the orm.
func (service *Service) BatchCreateGatewayStatus(batchGatewayStatus models.BatchGatewayStatus) []models.PersistedGatewayStatus {
	statusList := make([]models.PersistedGatewayStatus, len(batchGatewayStatus.Status))
	for i, gatewayStatus := range batchGatewayStatus.Status {
		persistedGatewayStatus := models.PersistedGatewayStatus{
			GatewayStatus: gatewayStatus,
			Timestamp: timemock.Now().UnixNano(),
		}
		statusList[i] = persistedGatewayStatus
	}

	service.db.BatchAddGatewayStatus(statusList)

	return statusList
}

// BatchGetGatewayStatusReport gets BatchGatewayStatusReport which contain multiple GatewayStatusReport.
func (service *Service) BatchGetGatewayStatusReport() models.BatchGatewayStatusReport {
	return service.db.LoadNonStaleGatewayReports()
}

// SaveBatchStatusReport builds and saves a status report for multiple gateways simultaneously.
// Those reports can be updated once whenever we receive a new status,
// and the saved results can then be queried. This keeps us from having to build the report dynamically
// on every request at runtime.
func (service *Service) SaveBatchGatewayStatusReport(status []models.PersistedGatewayStatus) models.BatchGatewayStatusReport {
	pubkeys := make([]string, len(status))
	for i := range status {
		pubkeys[i] = status[i].PubKey
	}
	batchReport := service.db.BatchLoadGatewayReports(pubkeys)

	// that's super crude but I don't think db results are guaranteed to come in order, plus some entries might
	// not exist
	reportMap := make(map[string]int)
	for i, report := range batchReport.Report {
		reportMap[report.PubKey] = i
	}

	for _, gatewayStatus := range status {
		if reportIdx, ok := reportMap[gatewayStatus.PubKey]; ok {
			service.updateGatewayReportUpToLastHour(&batchReport.Report[reportIdx], &gatewayStatus)
		} else {
			var freshReport models.GatewayStatusReport
			service.updateGatewayReportUpToLastHour(&freshReport, &gatewayStatus)
			batchReport.Report = append(batchReport.Report, freshReport)
			reportMap[freshReport.PubKey] = len(batchReport.Report) - 1
		}
	}

	service.db.SaveBatchGatewayStatusReport(batchReport)

	return batchReport
}

// SaveStatusReport builds and saves a status report for a gatewa. The report can be updated once
// whenever we receive a new status, and the saved result can then be queried. This keeps us from
// having to build the report dynamically on every request at runtime.
func (service *Service) SaveGatewayStatusReport(status models.PersistedGatewayStatus) models.GatewayStatusReport {
	report := service.db.LoadGatewayReport(status.PubKey)

	service.updateGatewayReportUpToLastHour(&report, &status)
	service.db.SaveGatewayStatusReport(report)

	return report
}

func (service *Service) updateGatewayReportUpToLastHour(report *models.GatewayStatusReport, status *models.PersistedGatewayStatus) {
	report.PubKey = status.PubKey // crude, we do this in case it's a fresh struct returned from the db
	report.Owner = status.Owner

	if status.IPVersion == "4" {
		report.MostRecentIPV4 = *status.Up
		report.Last5MinutesIPV4 = service.CalculateGatewayUptime(status.PubKey, "4", minutesAgo(5))
		report.LastHourIPV4 = service.CalculateGatewayUptime(status.PubKey, "4", minutesAgo(60))
	} else if status.IPVersion == "6" {
		report.MostRecentIPV6 = *status.Up
		report.Last5MinutesIPV6 = service.CalculateGatewayUptime(status.PubKey, "6", minutesAgo(5))
		report.LastHourIPV6 = service.CalculateGatewayUptime(status.PubKey, "6", minutesAgo(60))
	}
}

func (service *Service) CalculateGatewayUptime(pubkey string, ipVersion string, since int64) int {
	statuses := service.db.ListGatewayStatusSince(pubkey, ipVersion, since)
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

func minutesAgo(minutes int) int64 {
	now := timemock.Now()
	return now.Add(time.Duration(-minutes) * time.Minute).UnixNano()
}