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
	"github.com/nymtech/node-status-api/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path"
)

// IDb holds status information
type IDb interface {
	AddMixStatus(models.PersistedMixStatus)
	BatchAddMixStatus(status []models.PersistedMixStatus)
	ListMixStatus(pubkey string, limit int) []models.PersistedMixStatus
	ListMixStatusDateRange(pubkey string, ipVersion string, start int64, end int64) []models.PersistedMixStatus
	LoadMixReport(pubkey string) models.MixStatusReport
	LoadNonStaleMixReports() models.BatchMixStatusReport
	BatchLoadMixReports(pubkeys []string) models.BatchMixStatusReport
	BatchLoadAllMixReports() models.BatchMixStatusReport
	RemoveMixReports(pubkeys []string)
	SaveMixStatusReport(models.MixStatusReport)
	SaveBatchMixStatusReport(models.BatchMixStatusReport)

	ListMixStatusSince(pubkey string, ipVersion string, since int64) []models.PersistedMixStatus
	RemoveOldMixStatuses(before int64)
	GetActiveMixes(since int64) []string


	AddGatewayStatus(models.PersistedGatewayStatus)
	BatchAddGatewayStatus(status []models.PersistedGatewayStatus)
	ListGatewayStatus(pubkey string, limit int) []models.PersistedGatewayStatus
	ListGatewayStatusDateRange(pubkey string, ipVersion string, start int64, end int64) []models.PersistedGatewayStatus
	LoadGatewayReport(pubkey string) models.GatewayStatusReport
	LoadNonStaleGatewayReports() models.BatchGatewayStatusReport
	BatchLoadGatewayReports(pubkeys []string) models.BatchGatewayStatusReport
	SaveGatewayStatusReport(models.GatewayStatusReport)
	SaveBatchGatewayStatusReport(models.BatchGatewayStatusReport)

	ListGatewayStatusSince(pubkey string, ipVersion string, since int64) []models.PersistedGatewayStatus
	RemoveOldGatewayStatuses(before int64)
	GetActiveGateways(since int64) []string
}

const MaxReportSize = 2000
const MaxStatusesPerInsertion = 3000

// Db is a hashtable that holds mixnode uptime mixmining
type Db struct {
	orm *gorm.DB
}

// NewDb constructor
func NewDb(isTest bool) *Db {
	database, err := gorm.Open(sqlite.Open(dbPath(isTest)), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to orm!")
	}

	// mix status migration
	if err := database.AutoMigrate(&models.PersistedMixStatus{}); err != nil {
		log.Fatal(err)
	}
	if err := database.AutoMigrate(&models.MixStatusReport{}); err != nil {
		log.Fatal(err)
	}

	if err := database.AutoMigrate(&models.PersistedGatewayStatus{}); err != nil {
		log.Fatal(err)
	}
	if err := database.AutoMigrate(&models.GatewayStatusReport{}); err != nil {
		log.Fatal(err)
	}

	d := Db{
		database,
	}
	return &d
}

func dbPath(isTest bool) string {
	if isTest {
		db, err := ioutil.TempFile("", "test_mixmining.db")
		if err != nil {
			panic(err)
		}
		return db.Name()
	}

	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	dbPath := path.Join(usr.HomeDir, ".nym")
	if err := os.MkdirAll(dbPath, os.ModePerm); err != nil {
		log.Fatal(err)
	}
	db := path.Join(dbPath, "mixmining.db")
	fmt.Printf("db is: %s\n", db)
	return db
}

// Add saves a PersistedMixStatus
func (db *Db) AddMixStatus(status models.PersistedMixStatus) {
	db.orm.Create(status)
}

// If only there was some *GENERIC* way to not repeat this code...
func splitPersistedMixStatuses(statusList []models.PersistedMixStatus, chunkSize int) [][]models.PersistedMixStatus {
	dataCopy := make([]models.PersistedMixStatus, len(statusList))
	copy(dataCopy, statusList)

	var chunks [][]models.PersistedMixStatus
	for chunkSize < len(dataCopy) {
		dataCopy, chunks = dataCopy[chunkSize:], append(chunks, dataCopy[0:chunkSize:chunkSize])
	}

	return append(chunks, dataCopy)
}

// BatchAdd saves multiple PersistedMixStatus
func (db *Db) BatchAddMixStatus(status []models.PersistedMixStatus) {
	// with statuses > 7000 statuses I was getting `save error: too many SQL variables[GIN]` error so I had to split
	// the create operation
	for _, statusChunk := range splitPersistedMixStatuses(status, MaxStatusesPerInsertion) {
		db.orm.Create(statusChunk)
	}
}

// List returns all models.PersistedMixStatus in the orm
func (db *Db) ListMixStatus(pubkey string, limit int) []models.PersistedMixStatus {
	var statuses []models.PersistedMixStatus
	if err := db.orm.Order("timestamp desc").Limit(limit).Where("pub_key = ?", pubkey).Find(&statuses).Error; err != nil {
		return make([]models.PersistedMixStatus, 0)
	}
	return statuses
}

// ListDateRange lists all persisted mix statuses for a node for either IPv4 or IPv6 within the specified date range
func (db *Db) ListMixStatusDateRange(pubkey string, ipVersion string, start int64, end int64) []models.PersistedMixStatus {
	var statuses []models.PersistedMixStatus
	if err := db.orm.Order("timestamp desc").Where("pub_key = ?", pubkey).Where("ip_version = ?", ipVersion).Where("timestamp >= ?", start).Where("timestamp <= ?", end).Find(&statuses).Error; err != nil {
		return make([]models.PersistedMixStatus, 0)
	}
	return statuses
}

// ListMixStatusSinceWithLimit lists all persisted mix statuses for a node for either IPv4 or IPv6 since the specified timestamp with the maximum of `limit` results
func (db *Db) ListMixStatusSince(pubkey string, ipVersion string, since int64) []models.PersistedMixStatus {
	var statuses []models.PersistedMixStatus
	// resultant query:
	// SELECT * FROM (SELECT * FROM persisted_mix_statuses p WHERE p.pub_key = ? AND p.ip_version = ? AND p.timestamp >= ? ) ORDER BY timestamp desc;
	if err := db.orm.Table("(?)", db.orm.Model(&models.PersistedMixStatus{}).Where("pub_key = ?", pubkey).Where("ip_version = ?", ipVersion).Where("timestamp >= ?", since)).Order("timestamp desc").Find(&statuses).Error; err != nil {
		return make([]models.PersistedMixStatus, 0)
	}
	return statuses
}

// RemoveOldStatuses removes all `PersistedMixStatus` that were created before the provided timestamp.
func (db *Db) RemoveOldMixStatuses(before int64) {
	if err := db.orm.Unscoped().Where("timestamp < ?", before).Delete(&models.PersistedMixStatus{}).Error; err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to remove old statuses from the database - %v\n", err)
	}
}

// SaveMixStatusReport creates or updates a status summary report for a given mixnode in the database
func (db *Db) SaveMixStatusReport(report models.MixStatusReport) {
	create := db.orm.Save(report)
	if create.Error != nil {
		fmt.Printf("Mix status report creation error: %+v", create.Error)
	}
}

// SaveBatchMixStatusReport creates or updates a status summary report for multiple mixnodes in the database
func (db *Db) SaveBatchMixStatusReport(report models.BatchMixStatusReport) {
	// with statuses of > 3500 nodes I was getting `save error: too many SQL variables[GIN]` error so I had to split
	// the save operation
	save := func (db *Db, report models.BatchMixStatusReport) {
		if len(report.Report) == 0 {
			return
		}
		if result := db.orm.Save(report.Report); result.Error != nil {
			fmt.Printf("Batch Mix status report save error: %+v", result.Error)
		}
	}

	if len(report.Report) < MaxReportSize {
		save(db, report)
	} else {
		chunks := report.SplitToChunks(MaxReportSize)
		for _, reportChunk := range chunks {
			save(db, reportChunk)
		}
	}
}

// LoadReport retrieves a models.MixStatusReport.
// If a report isn't found, it crudely generates a new instance and returns that instead.
func (db *Db) LoadMixReport(pubkey string) models.MixStatusReport {
	var report models.MixStatusReport

	if retrieve := db.orm.First(&report, "pub_key = ?", pubkey); retrieve.Error != nil {
		fmt.Printf("ERROR while retrieving mix status report %+v", retrieve.Error)
		return models.MixStatusReport{}
	}
	return report
}

// LoadNonStaleReports retrieves a models.BatchMixStatusReport, such that each mixnode
// in the retrieved report must have been online for at least a single measurement in the last day.
// If a report isn't found, it crudely generates a new instance and returns that instead.
func (db *Db) LoadNonStaleMixReports() models.BatchMixStatusReport {
	var reports []models.MixStatusReport

	if retrieve := db.orm.Where("last_day_ip_v4 > 0").Or("last_day_ip_v6 > 0").Find(&reports); retrieve.Error != nil {
		fmt.Printf("ERROR while retrieving multiple mix status report %+v", retrieve.Error)
		return models.BatchMixStatusReport{Report: make([]models.MixStatusReport, 0)}
	}
	return models.BatchMixStatusReport{Report: reports}
}

// BatchLoadReports retrieves a models.BatchMixStatusReport based on provided set of public keys.
// If a report isn't found, it crudely generates a new instance and returns that instead.
func (db *Db) BatchLoadMixReports(pubkeys []string) models.BatchMixStatusReport {
	var reports []models.MixStatusReport

	if retrieve := db.orm.Where("pub_key IN ?", pubkeys).Find(&reports); retrieve.Error != nil {
		fmt.Printf("ERROR while retrieving multiple mix status report %+v", retrieve.Error)
		return models.BatchMixStatusReport{Report: make([]models.MixStatusReport, 0)}
	}
	return models.BatchMixStatusReport{Report: reports}
}

// BatchLoadAllMixReports retrieves a models.BatchMixStatusReport containing data of all nodes
func (db *Db) BatchLoadAllMixReports() models.BatchMixStatusReport {
	var reports []models.MixStatusReport

	if retrieve := db.orm.Find(&reports); retrieve.Error != nil {
		fmt.Printf("ERROR while retrieving all mix status report %+v", retrieve.Error)
		return models.BatchMixStatusReport{Report: make([]models.MixStatusReport, 0)}
	}
	return models.BatchMixStatusReport{Report: reports}
}

// RemoveMixReports removes MixReports of nodes specified by the provided public keys.
func (db *Db) RemoveMixReports(pubkeys []string) {
	if err := db.orm.Unscoped().Where("pub_key IN ?", pubkeys).Delete(&models.MixStatusReport{}).Error; err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to remove old reports from the database - %v\n", err)
	}
}

func (db *Db) GetActiveMixes(since int64) []string {
	var reports []models.PersistedMixStatus

	if err := db.orm.Select("pub_key").Where("timestamp > ?", since).Group("pub_key").Find(&reports).Error; err != nil {
		fmt.Printf("ERROR while retrieving currently active nodes %+v", err)
		return []string{}
	}

	keys := make([]string, len(reports))
	for i, report := range reports {
		keys[i] = report.PubKey
	}

	return keys
}



















// Add saves a PersistedGatewayStatus
func (db *Db) AddGatewayStatus(status models.PersistedGatewayStatus) {
	db.orm.Create(status)
}

// If only there was some *GENERIC* way to not repeat this code...
func splitPersistedGatewayStatuses(statusList []models.PersistedGatewayStatus, chunkSize int) [][]models.PersistedGatewayStatus {
	dataCopy := make([]models.PersistedGatewayStatus, len(statusList))
	copy(dataCopy, statusList)

	var chunks [][]models.PersistedGatewayStatus
	for chunkSize < len(dataCopy) {
		dataCopy, chunks = dataCopy[chunkSize:], append(chunks, dataCopy[0:chunkSize:chunkSize])
	}

	return append(chunks, dataCopy)
}

// BatchAdd saves multiple PersistedGatewayStatus
func (db *Db) BatchAddGatewayStatus(status []models.PersistedGatewayStatus) {
	// with statuses > 7000 statuses I was getting `save error: too many SQL variables[GIN]` error so I had to split
	// the create operation
	for _, statusChunk := range splitPersistedGatewayStatuses(status, MaxStatusesPerInsertion) {
		db.orm.Create(statusChunk)
	}
}

// List returns all models.PersistedGatewayStatus in the orm
func (db *Db) ListGatewayStatus(pubkey string, limit int) []models.PersistedGatewayStatus {
	var statuses []models.PersistedGatewayStatus
	if err := db.orm.Order("timestamp desc").Limit(limit).Where("pub_key = ?", pubkey).Find(&statuses).Error; err != nil {
		return make([]models.PersistedGatewayStatus, 0)
	}
	return statuses
}

// ListDateRange lists all persisted gateway statuses for a node for either IPv4 or IPv6 within the specified date range
func (db *Db) ListGatewayStatusDateRange(pubkey string, ipVersion string, start int64, end int64) []models.PersistedGatewayStatus {
	var statuses []models.PersistedGatewayStatus
	if err := db.orm.Order("timestamp desc").Where("pub_key = ?", pubkey).Where("ip_version = ?", ipVersion).Where("timestamp >= ?", start).Where("timestamp <= ?", end).Find(&statuses).Error; err != nil {
		return make([]models.PersistedGatewayStatus, 0)
	}
	return statuses
}

// ListGatewayStatusSinceWithLimit lists all persisted gateway statuses for a node for either IPv4 or IPv6 since the specified timestamp with the maximum of `limit` results
func (db *Db) ListGatewayStatusSince(pubkey string, ipVersion string, since int64) []models.PersistedGatewayStatus {
	var statuses []models.PersistedGatewayStatus
	// resultant query:
	// SELECT * FROM (SELECT * FROM persisted_gateway_statuses p WHERE p.pub_key = ? AND p.ip_version = ? AND p.timestamp >= ? ) ORDER BY timestamp desc;
	if err := db.orm.Table("(?)", db.orm.Model(&models.PersistedGatewayStatus{}).Where("pub_key = ?", pubkey).Where("ip_version = ?", ipVersion).Where("timestamp >= ?", since)).Order("timestamp desc").Find(&statuses).Error; err != nil {
		return make([]models.PersistedGatewayStatus, 0)
	}
	return statuses
}

// RemoveOldStatuses removes all `PersistedGatewayStatus` that were created before the provided timestamp.
func (db *Db) RemoveOldGatewayStatuses(before int64) {
	if err := db.orm.Unscoped().Where("timestamp < ?", before).Delete(&models.PersistedGatewayStatus{}).Error; err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to remove old statuses from the database - %v\n", err)
	}
}

// SaveGatewayStatusReport creates or updates a status summary report for a given gateway in the database
func (db *Db) SaveGatewayStatusReport(report models.GatewayStatusReport) {
	create := db.orm.Save(report)
	if create.Error != nil {
		fmt.Printf("Gateway status report creation error: %+v", create.Error)
	}
}

// SaveBatchGatewayStatusReport creates or updates a status summary report for multiple mixnodes in the database
func (db *Db) SaveBatchGatewayStatusReport(report models.BatchGatewayStatusReport) {
	// with statuses of > 3500 nodes I was getting `save error: too many SQL variables[GIN]` error so I had to split
	// the save operation
	save := func (db *Db, report models.BatchGatewayStatusReport) {
		if len(report.Report) == 0 {
			return
		}
		if result := db.orm.Save(report.Report); result.Error != nil {
			fmt.Printf("Batch Gateway status report save error: %+v", result.Error)
		}
	}

	if len(report.Report) < MaxReportSize {
		save(db, report)
	} else {
		chunks := report.SplitToChunks(MaxReportSize)
		for _, reportChunk := range chunks {
			save(db, reportChunk)
		}
	}
}

// LoadReport retrieves a models.GatewayStatusReport.
// If a report isn't found, it crudely generates a new instance and returns that instead.
func (db *Db) LoadGatewayReport(pubkey string) models.GatewayStatusReport {
	var report models.GatewayStatusReport

	if retrieve := db.orm.First(&report, "pub_key = ?", pubkey); retrieve.Error != nil {
		fmt.Printf("ERROR while retrieving mix status report %+v", retrieve.Error)
		return models.GatewayStatusReport{}
	}
	return report
}

// LoadNonStaleReports retrieves a models.BatchGatewayStatusReport, such that each gateway
// in the retrieved report must have been online for at least a single measurement in the last day.
// If a report isn't found, it crudely generates a new instance and returns that instead.
func (db *Db) LoadNonStaleGatewayReports() models.BatchGatewayStatusReport {
	var reports []models.GatewayStatusReport

	if retrieve := db.orm.Where("last_day_ip_v4 > 0").Or("last_day_ip_v6 > 0").Find(&reports); retrieve.Error != nil {
		fmt.Printf("ERROR while retrieving multiple gateway status report %+v", retrieve.Error)
		return models.BatchGatewayStatusReport{Report: make([]models.GatewayStatusReport, 0)}
	}
	return models.BatchGatewayStatusReport{Report: reports}
}

// BatchLoadReports retrieves a models.BatchGatewayStatusReport based on provided set of public keys.
// If a report isn't found, it crudely generates a new instance and returns that instead.
func (db *Db) BatchLoadGatewayReports(pubkeys []string) models.BatchGatewayStatusReport {
	var reports []models.GatewayStatusReport

	if retrieve := db.orm.Where("pub_key IN ?", pubkeys).Find(&reports); retrieve.Error != nil {
		fmt.Printf("ERROR while retrieving multiple gatweway status report %+v", retrieve.Error)
		return models.BatchGatewayStatusReport{Report: make([]models.GatewayStatusReport, 0)}
	}
	return models.BatchGatewayStatusReport{Report: reports}
}

func (db *Db) GetActiveGateways(since int64) []string {
	var reports []models.PersistedGatewayStatus

	if err := db.orm.Select("pub_key").Where("timestamp > ?", since).Group("pub_key").Find(&reports).Error; err != nil {
		fmt.Printf("ERROR while retrieving currently active nodes %+v", err)
		return []string{}
	}

	keys := make([]string, len(reports))
	for i, report := range reports {
		keys[i] = report.PubKey
	}

	return keys
}
