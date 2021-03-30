// Code generated by mockery v2.7.4. DO NOT EDIT.

package mocks

import (
	models "github.com/nymtech/node-status-api/models"
	mock "github.com/stretchr/testify/mock"
)

// IService is an autogenerated mock type for the IService type
type IService struct {
	mock.Mock
}

// BatchCreateMixStatus provides a mock function with given fields: batchMixStatus
func (_m *IService) BatchCreateMixStatus(batchMixStatus models.BatchMixStatus) []models.PersistedMixStatus {
	ret := _m.Called(batchMixStatus)

	var r0 []models.PersistedMixStatus
	if rf, ok := ret.Get(0).(func(models.BatchMixStatus) []models.PersistedMixStatus); ok {
		r0 = rf(batchMixStatus)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.PersistedMixStatus)
		}
	}

	return r0
}

// BatchGetMixStatusReport provides a mock function with given fields:
func (_m *IService) BatchGetMixStatusReport() models.BatchMixStatusReport {
	ret := _m.Called()

	var r0 models.BatchMixStatusReport
	if rf, ok := ret.Get(0).(func() models.BatchMixStatusReport); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(models.BatchMixStatusReport)
	}

	return r0
}

// CreateMixStatus provides a mock function with given fields: mixStatus
func (_m *IService) CreateMixStatus(mixStatus models.MixStatus) models.PersistedMixStatus {
	ret := _m.Called(mixStatus)

	var r0 models.PersistedMixStatus
	if rf, ok := ret.Get(0).(func(models.MixStatus) models.PersistedMixStatus); ok {
		r0 = rf(mixStatus)
	} else {
		r0 = ret.Get(0).(models.PersistedMixStatus)
	}

	return r0
}

// GetStatusReport provides a mock function with given fields: pubkey
func (_m *IService) GetStatusReport(pubkey string) models.MixStatusReport {
	ret := _m.Called(pubkey)

	var r0 models.MixStatusReport
	if rf, ok := ret.Get(0).(func(string) models.MixStatusReport); ok {
		r0 = rf(pubkey)
	} else {
		r0 = ret.Get(0).(models.MixStatusReport)
	}

	return r0
}

// ListMixStatus provides a mock function with given fields: pubkey
func (_m *IService) ListMixStatus(pubkey string) []models.PersistedMixStatus {
	ret := _m.Called(pubkey)

	var r0 []models.PersistedMixStatus
	if rf, ok := ret.Get(0).(func(string) []models.PersistedMixStatus); ok {
		r0 = rf(pubkey)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]models.PersistedMixStatus)
		}
	}

	return r0
}

// SaveBatchStatusReport provides a mock function with given fields: status
func (_m *IService) SaveBatchStatusReport(status []models.PersistedMixStatus) models.BatchMixStatusReport {
	ret := _m.Called(status)

	var r0 models.BatchMixStatusReport
	if rf, ok := ret.Get(0).(func([]models.PersistedMixStatus) models.BatchMixStatusReport); ok {
		r0 = rf(status)
	} else {
		r0 = ret.Get(0).(models.BatchMixStatusReport)
	}

	return r0
}

// SaveStatusReport provides a mock function with given fields: status
func (_m *IService) SaveStatusReport(status models.PersistedMixStatus) models.MixStatusReport {
	ret := _m.Called(status)

	var r0 models.MixStatusReport
	if rf, ok := ret.Get(0).(func(models.PersistedMixStatus) models.MixStatusReport); ok {
		r0 = rf(status)
	} else {
		r0 = ret.Get(0).(models.MixStatusReport)
	}

	return r0
}
