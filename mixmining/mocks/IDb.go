// Code generated by mockery v2.7.4. DO NOT EDIT.

package mocks

import (
	models "github.com/nymtech/node-status-api/models"
	mock "github.com/stretchr/testify/mock"
)

// IDb is an autogenerated mock type for the IDb type
type IDb struct {
	mock.Mock
}

// AddMixStatus provides a mock function with given fields: _a0
func (_m *IDb) AddMixStatus(_a0 models.PersistedMixStatus) {
	_m.Called(_a0)
}

// BatchAddMixStatus provides a mock function with given fields: status
func (_m *IDb) BatchAddMixStatus(status []models.PersistedMixStatus) {
	_m.Called(status)
}

// BatchLoadReports provides a mock function with given fields: pubkeys
func (_m *IDb) BatchLoadReports(pubkeys []string) models.BatchMixStatusReport {
	ret := _m.Called(pubkeys)

	var r0 models.BatchMixStatusReport
	if rf, ok := ret.Get(0).(func([]string) models.BatchMixStatusReport); ok {
		r0 = rf(pubkeys)
	} else {
		r0 = ret.Get(0).(models.BatchMixStatusReport)
	}

	return r0
}

// GetActiveNodes provides a mock function with given fields: since
func (_m *IDb) GetActiveNodes(since int64) []string {
	ret := _m.Called(since)

	var r0 []string
	if rf, ok := ret.Get(0).(func(int64) []string); ok {
		r0 = rf(since)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	return r0
}

// ListMixStatus provides a mock function with given fields: pubkey, limit
func (_m *IDb) ListMixStatus(pubkey string, limit int) []models.PersistedMixStatus {
	ret := _m.Called(pubkey, limit)

	var r0 []models.PersistedMixStatus
	if rf, ok := ret.Get(0).(func(string, int) []models.PersistedMixStatus); ok {
		r0 = rf(pubkey, limit)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.PersistedMixStatus)
		}
	}

	return r0
}

// ListMixStatusDateRange provides a mock function with given fields: pubkey, ipVersion, start, end
func (_m *IDb) ListMixStatusDateRange(pubkey string, ipVersion string, start int64, end int64) []models.PersistedMixStatus {
	ret := _m.Called(pubkey, ipVersion, start, end)

	var r0 []models.PersistedMixStatus
	if rf, ok := ret.Get(0).(func(string, string, int64, int64) []models.PersistedMixStatus); ok {
		r0 = rf(pubkey, ipVersion, start, end)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.PersistedMixStatus)
		}
	}

	return r0
}

// ListMixStatusSince provides a mock function with given fields: pubkey, ipVersion, since
func (_m *IDb) ListMixStatusSince(pubkey string, ipVersion string, since int64) []models.PersistedMixStatus {
	ret := _m.Called(pubkey, ipVersion, since)

	var r0 []models.PersistedMixStatus
	if rf, ok := ret.Get(0).(func(string, string, int64) []models.PersistedMixStatus); ok {
		r0 = rf(pubkey, ipVersion, since)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.PersistedMixStatus)
		}
	}

	return r0
}

// LoadNonStaleReports provides a mock function with given fields:
func (_m *IDb) LoadNonStaleReports() models.BatchMixStatusReport {
	ret := _m.Called()

	var r0 models.BatchMixStatusReport
	if rf, ok := ret.Get(0).(func() models.BatchMixStatusReport); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(models.BatchMixStatusReport)
	}

	return r0
}

// LoadReport provides a mock function with given fields: pubkey
func (_m *IDb) LoadReport(pubkey string) models.MixStatusReport {
	ret := _m.Called(pubkey)

	var r0 models.MixStatusReport
	if rf, ok := ret.Get(0).(func(string) models.MixStatusReport); ok {
		r0 = rf(pubkey)
	} else {
		r0 = ret.Get(0).(models.MixStatusReport)
	}

	return r0
}

// RemoveOldStatuses provides a mock function with given fields: before
func (_m *IDb) RemoveOldStatuses(before int64) {
	_m.Called(before)
}

// SaveBatchMixStatusReport provides a mock function with given fields: _a0
func (_m *IDb) SaveBatchMixStatusReport(_a0 models.BatchMixStatusReport) {
	_m.Called(_a0)
}

// SaveMixStatusReport provides a mock function with given fields: _a0
func (_m *IDb) SaveMixStatusReport(_a0 models.MixStatusReport) {
	_m.Called(_a0)
}
