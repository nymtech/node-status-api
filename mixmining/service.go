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
	SaveStatusReport(status models.PersistedMixStatus) models.MixStatusReport
	GetStatusReport(pubkey string) models.MixStatusReport

	SaveBatchStatusReport(status []models.PersistedMixStatus) models.BatchMixStatusReport
	BatchCreateMixStatus(batchMixStatus models.BatchMixStatus) []models.PersistedMixStatus
	BatchGetMixStatusReport() models.BatchMixStatusReport
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
		service.updateLastDayReports()
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

func (service *Service) updateLastDayReports() models.BatchMixStatusReport {
	dayAgo := timemock.Now().Add(-time.Hour * 24).UnixNano()
	allActive := service.db.GetActiveNodes(dayAgo)

	batchReport := service.db.BatchLoadReports(allActive)

	for i := range batchReport.Report {
		lastDayUptime := service.CalculateUptime(batchReport.Report[i].PubKey, "4", dayAgo)
		if lastDayUptime == -1 {
			// there were no reports to calculate uptime with
			// but this should NEVER happen as we only loaded reports for the nodes that received status data
			// in last 24h
			continue
		}

		batchReport.Report[i].LastDayIPV4 = lastDayUptime
		batchReport.Report[i].LastDayIPV6 = service.CalculateUptime(batchReport.Report[i].PubKey, "6", dayAgo)
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
	for i, report := range batchReport.Report {
		reportMap[report.PubKey] = i
	}

	for _, mixStatus := range status {
		if reportIdx, ok := reportMap[mixStatus.PubKey]; ok {
			service.updateReportUpToLastHour(&batchReport.Report[reportIdx], &mixStatus)
		} else {
			var freshReport models.MixStatusReport
			service.updateReportUpToLastHour(&freshReport, &mixStatus)
			batchReport.Report = append(batchReport.Report, freshReport)
			reportMap[freshReport.PubKey] = len(batchReport.Report) - 1
		}
	}

	service.db.SaveBatchMixStatusReport(batchReport)

	return batchReport
}

func (service *Service) updateReportUpToLastHour(report *models.MixStatusReport, status *models.PersistedMixStatus) {
	report.PubKey = status.PubKey // crude, we do this in case it's a fresh struct returned from the db
	report.Owner = status.Owner
	
	if status.IPVersion == "4" {
		report.MostRecentIPV4 = *status.Up
		report.Last5MinutesIPV4 = service.CalculateUptime(status.PubKey, "4", minutesAgo(5))
		report.LastHourIPV4 = service.CalculateUptime(status.PubKey, "4", minutesAgo(60))
	} else if status.IPVersion == "6" {
		report.MostRecentIPV6 = *status.Up
		report.Last5MinutesIPV6 = service.CalculateUptime(status.PubKey, "6", minutesAgo(5))
		report.LastHourIPV6 = service.CalculateUptime(status.PubKey, "6", minutesAgo(60))
	}
}

// SaveStatusReport builds and saves a status report for a mixnode. The report can be updated once
// whenever we receive a new status, and the saved result can then be queried. This keeps us from
// having to build the report dynamically on every request at runtime.
func (service *Service) SaveStatusReport(status models.PersistedMixStatus) models.MixStatusReport {
	report := service.db.LoadReport(status.PubKey)

	service.updateReportUpToLastHour(&report, &status)
	service.db.SaveMixStatusReport(report)

	return report
}

func (service *Service) CalculateUptime(pubkey string, ipVersion string, since int64) int {
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

func (service *Service) calculatePercent(num int, outOf int) int {
	return int(float32(num) / float32(outOf) * 100)
}

func minutesAgo(minutes int) int64 {
	now := timemock.Now()
	return now.Add(time.Duration(-minutes) * time.Minute).UnixNano()
}