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
	LoadReport(pubkey string) models.MixStatusReport
	LoadNonStaleReports() models.BatchMixStatusReport
	BatchLoadReports(pubkeys []string) models.BatchMixStatusReport
	SaveMixStatusReport(models.MixStatusReport)
	SaveBatchMixStatusReport(models.BatchMixStatusReport)

	ListMixStatusSince(pubkey string, ipVersion string, since int64) []models.PersistedMixStatus
	RemoveOldStatuses(before int64)
	GetActiveNodes(since int64) []string
}

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

// BatchAdd saves multiple PersistedMixStatus
func (db *Db) BatchAddMixStatus(status []models.PersistedMixStatus) {
	db.orm.Create(status)
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
func (db *Db) RemoveOldStatuses(before int64) {
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

// SaveBatchMixStatusReport creates or updates a status summary report for multiple mixnodex in the database
func (db *Db) SaveBatchMixStatusReport(report models.BatchMixStatusReport) {
	if result := db.orm.Save(report.Report); result.Error != nil {
		fmt.Printf("Batch Mix status report save error: %+v", result.Error)
	}
}

// LoadReport retrieves a models.MixStatusReport.
// If a report isn't found, it crudely generates a new instance and returns that instead.
func (db *Db) LoadReport(pubkey string) models.MixStatusReport {
	var report models.MixStatusReport

	if retrieve := db.orm.First(&report, "pub_key = ?", pubkey); retrieve.Error != nil {
		fmt.Printf("ERROR while retrieving mix status report %+v", retrieve.Error)
		return models.MixStatusReport{}
	}
	return report
}

// LoadNonStaleReports retrieves a models.BatchMixStatusReport, such that each mixnode
// in the retrieved report must have been online for over 50% of time in the last day.
// If a report isn't found, it crudely generates a new instance and returns that instead.
func (db *Db) LoadNonStaleReports() models.BatchMixStatusReport {
	var reports []models.MixStatusReport

	if retrieve := db.orm.Where("last_day_ip_v4 >= 50").Or("last_day_ip_v6 >= 50").Find(&reports); retrieve.Error != nil {
		fmt.Printf("ERROR while retrieving multiple mix status report %+v", retrieve.Error)
		return models.BatchMixStatusReport{Report: make([]models.MixStatusReport, 0)}
	}
	return models.BatchMixStatusReport{Report: reports}
}

// BatchLoadReports retrieves a models.BatchMixStatusReport based on provided set of public keys.
// If a report isn't found, it crudely generates a new instance and returns that instead.
func (db *Db) BatchLoadReports(pubkeys []string) models.BatchMixStatusReport {
	var reports []models.MixStatusReport

	if retrieve := db.orm.Where("pub_key IN ?", pubkeys).Find(&reports); retrieve.Error != nil {
		fmt.Printf("ERROR while retrieving multiple mix status report %+v", retrieve.Error)
		return models.BatchMixStatusReport{Report: make([]models.MixStatusReport, 0)}
	}
	return models.BatchMixStatusReport{Report: reports}
}

func (db *Db) GetActiveNodes(since int64) []string {
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
