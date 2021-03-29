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
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/nymtech/nym/validator/nym/models"

	"github.com/gin-gonic/gin"
	"github.com/nymtech/nym/validator/nym/mixmining/fixtures"
	"github.com/nymtech/nym/validator/nym/mixmining/mocks"
	. "github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
)

var _ = Describe("Controller", func() {
	Describe("creating a mix status", func() {
		Context("from a host other than localhost", func() {
			It("should fail", func() {
				router, _, _, _, _ := SetupRouter()
				badJSON, _ := json.Marshal(fixtures.XSSMixStatus())
				resp := performNonLocalRequest(router, "POST", "/api/mixmining", badJSON)
				assert.Equal(GinkgoT(), 403, resp.Result().StatusCode)
			})
		})

		Context("that has 'false' set for 'Up'", func() {
			It("should save the mix status", func() {
				boolfalse := false
				router, mockService, mockSanitizer, _, _ := SetupRouter()
				status := fixtures.GoodMixStatus()
				status.Up = &boolfalse

				savedStatus := fixtures.GoodPersistedMixStatus()
				savedStatus.Up = &boolfalse

				mockSanitizer.On("Sanitize", status).Return(status)
				mockService.On("CreateMixStatus", status).Return(savedStatus)
				mockService.On("SaveStatusReport", savedStatus).Return(models.MixStatusReport{})

				falseJSON, _ := json.Marshal(status)
				resp := performLocalHostRequest(router, "POST", "/api/mixmining", falseJSON)
				assert.Equal(GinkgoT(), 201, resp.Code)

			})
		})

		Context("containing xss", func() {
			It("should strip the xss attack, save the individual mix status, and update the status report for the given node", func() {
				router, mockService, mockSanitizer, _, _ := SetupRouter()

				mockSanitizer.On("Sanitize", fixtures.XSSMixStatus()).Return(fixtures.GoodMixStatus())
				mockService.On("CreateMixStatus", fixtures.GoodMixStatus()).Return(fixtures.GoodPersistedMixStatus())
				mockService.On("SaveStatusReport", fixtures.GoodPersistedMixStatus()).Return(models.MixStatusReport{})
				badJSON, _ := json.Marshal(fixtures.XSSMixStatus())

				resp := performLocalHostRequest(router, "POST", "/api/mixmining", badJSON)
				var response map[string]string
				json.Unmarshal([]byte(resp.Body.String()), &response)

				assert.Equal(GinkgoT(), 201, resp.Code)
				mockSanitizer.AssertCalled(GinkgoT(), "Sanitize", fixtures.XSSMixStatus())
				mockService.AssertCalled(GinkgoT(), "CreateMixStatus", fixtures.GoodMixStatus())
			})
		})
	})

	Describe("retrieving a mix status report (overview)", func() {
		Context("when a report does not yet exist", func() {
			It("should 404", func() {
				router, mockService, _, _, _ := SetupRouter()
				mockService.On("GetStatusReport", fixtures.MixStatusReport().PubKey).Return(models.MixStatusReport{})
				resp := performLocalHostRequest(router, "GET", "/api/mixmining/node/key1/report", nil)
				assert.Equal(GinkgoT(), 404, resp.Result().StatusCode)
			})
		})

		Context("when a report exists", func() {
			It("should return the report", func() {
				router, mockService, _, _, _ := SetupRouter()
				mockService.On("GetStatusReport", fixtures.MixStatusReport().PubKey).Return(fixtures.MixStatusReport())
				resp := performLocalHostRequest(router, "GET", "/api/mixmining/node/key1/report", nil)
				var response models.MixStatusReport
				json.Unmarshal([]byte(resp.Body.String()), &response)
				assert.Equal(GinkgoT(), 200, resp.Result().StatusCode)
				assert.Equal(GinkgoT(), fixtures.MixStatusReport(), response)
			})
		})
	})

	Describe("listing statuses for a node", func() {
		Context("when no statuses have yet been saved", func() {
			It("returns an empty list", func() {
				router, mockService, _, _, _ := SetupRouter()
				mockService.On("ListMixStatus", "foo").Return([]models.PersistedMixStatus{})
				resp := performLocalHostRequest(router, "GET", "/api/mixmining/node/foo/history", nil)

				assert.Equal(GinkgoT(), 200, resp.Code)
			})
		})
		Context("when some statuses exist", func() {
			It("should return the list of statuses as json", func() {
				router, mockService, _, _, _ := SetupRouter()
				mockService.On("ListMixStatus", "pubkey1").Return(fixtures.MixStatusesList())
				url := "/api/mixmining/node/pubkey1/history"
				resp := performLocalHostRequest(router, "GET", url, nil)
				var response []models.PersistedMixStatus
				json.Unmarshal([]byte(resp.Body.String()), &response)

				assert.Equal(GinkgoT(), 200, resp.Code)
				assert.Equal(GinkgoT(), fixtures.MixStatusesList(), response)
			})
		})
	})

	Describe("Creating batch mix status", func() {
		Context("from a host other than localhost", func() {
			It("should fail", func() {
				router, _, _, _, _ := SetupRouter()
				goodJSON, _ := json.Marshal(fixtures.GoodBatchMixStatus())
				resp := performNonLocalRequest(router, "POST", "/api/mixmining/batch", goodJSON)
				assert.Equal(GinkgoT(), 403, resp.Result().StatusCode)
			})
		})

		Context("Containing single status", func() {
			Context("that has 'false' set for 'Up'", func() {
				It("should save the mix status", func() {
					boolfalse := false
					router, mockService, _, _, mockBatchSanitizer := SetupRouter()
					singleStatusBatch := models.BatchMixStatus{Status: []models.MixStatus{fixtures.GoodMixStatus()}}
					singleStatusBatch.Status[0].Up = &boolfalse

					savedStatus := []models.PersistedMixStatus{{MixStatus: fixtures.GoodMixStatus(), Timestamp: 1234}}
					savedStatus[0].Up = &boolfalse

					mockBatchSanitizer.On("Sanitize", singleStatusBatch).Return(singleStatusBatch)
					mockService.On("BatchCreateMixStatus", singleStatusBatch).Return(savedStatus)
					mockService.On("SaveBatchStatusReport", savedStatus).Return(models.BatchMixStatusReport{Report: []models.MixStatusReport{}})

					falseJSON, _ := json.Marshal(singleStatusBatch)
					resp := performLocalHostRequest(router, "POST", "/api/mixmining/batch", falseJSON)

					assert.Equal(GinkgoT(), 201, resp.Code)
				})
			})

			Context("containing xss", func() {
				It("should strip the xss attack, save the individual mix status, and update the status report for the given node", func() {
					router, mockService, _, _, mockBatchSanitizer := SetupRouter()
					singleXSSStatusBatch := models.BatchMixStatus{Status: []models.MixStatus{fixtures.XSSMixStatus()}}
					singleStatusBatch := models.BatchMixStatus{Status: []models.MixStatus{fixtures.GoodMixStatus()}}
					savedStatus := []models.PersistedMixStatus{{MixStatus: fixtures.GoodMixStatus(), Timestamp: 1234}}

					mockBatchSanitizer.On("Sanitize", singleXSSStatusBatch).Return(singleStatusBatch)
					mockService.On("BatchCreateMixStatus", singleStatusBatch).Return(savedStatus)
					mockService.On("SaveBatchStatusReport", savedStatus).Return(models.BatchMixStatusReport{Report: []models.MixStatusReport{}})
					badJSON, _ := json.Marshal(singleXSSStatusBatch)

					resp := performLocalHostRequest(router, "POST", "/api/mixmining/batch", badJSON)
					var response map[string]string
					json.Unmarshal([]byte(resp.Body.String()), &response)

					assert.Equal(GinkgoT(), 201, resp.Code)
					mockBatchSanitizer.AssertCalled(GinkgoT(), "Sanitize", singleXSSStatusBatch)
					mockService.AssertCalled(GinkgoT(), "BatchCreateMixStatus", singleStatusBatch)
				})
			})
		})

		Context("Containing multiple status data", func() {
			Context("containing xss", func() {
				It("should strip the xss attack, save the individual mix status, and update the status report for the given node", func() {
					router, mockService, _, _, mockBatchSanitizer := SetupRouter()

					mockBatchSanitizer.On("Sanitize", fixtures.XSSBatchMixStatus()).Return(fixtures.GoodBatchMixStatus())
					mockService.On("BatchCreateMixStatus", fixtures.GoodBatchMixStatus()).Return(fixtures.GoodPersistedBatchMixStatus())
					mockService.On("SaveBatchStatusReport", fixtures.GoodPersistedBatchMixStatus()).Return(models.BatchMixStatusReport{Report: []models.MixStatusReport{}})
					badJSON, _ := json.Marshal(fixtures.XSSBatchMixStatus())

					resp := performLocalHostRequest(router, "POST", "/api/mixmining/batch", badJSON)
					var response map[string]string
					json.Unmarshal([]byte(resp.Body.String()), &response)

					assert.Equal(GinkgoT(), 201, resp.Code)
					mockBatchSanitizer.AssertCalled(GinkgoT(), "Sanitize", fixtures.XSSBatchMixStatus())
					mockService.AssertCalled(GinkgoT(), "BatchCreateMixStatus", fixtures.GoodBatchMixStatus())
				})
			})
		})

	})

	Describe("Retrieving full batch mix status report", func() {
		Context("when no reports exist yet", func() {
			It("should return empty report", func() {
				router, mockService, _, _, _ := SetupRouter()
				mockService.On("BatchGetMixStatusReport").Return(models.BatchMixStatusReport{Report: []models.MixStatusReport{}})
				resp := performLocalHostRequest(router, "GET", "/api/mixmining/fullreport", nil)
				assert.Equal(GinkgoT(), 200, resp.Result().StatusCode)

				var response models.BatchMixStatusReport
				json.Unmarshal([]byte(resp.Body.String()), &response)

				assert.Equal(GinkgoT(), len(response.Report), 0)
			})
		})

		Context("when a report exists", func() {
			It("should return the report", func() {
				router, mockService, _, _, _ := SetupRouter()
				reqReport := models.BatchMixStatusReport{Report: []models.MixStatusReport{fixtures.MixStatusReport()}}
				mockService.On("BatchGetMixStatusReport").Return(reqReport)
				resp := performLocalHostRequest(router, "GET", "/api/mixmining/fullreport", nil)
				var response models.BatchMixStatusReport
				json.Unmarshal([]byte(resp.Body.String()), &response)
				assert.Equal(GinkgoT(), 200, resp.Result().StatusCode)
				assert.Equal(GinkgoT(), reqReport, response)
			})
		})
	})
})

func SetupRouter() (*gin.Engine, *mocks.IService, *mocks.Sanitizer, *mocks.GenericSanitizer, *mocks.BatchSanitizer) {
	mockSanitizer := new(mocks.Sanitizer)
	mockBatchSanitizer := new(mocks.BatchSanitizer)
	mockGenericSanitizer := new(mocks.GenericSanitizer)
	mockService := new(mocks.IService)

	// on startup there will be no nodes
	mockService.On("MixCount").Return(0)
	mockService.On("GatewayCount").Return(0)
	mockService.On("StartupPurge")

	cfg := Config{
		BatchSanitizer:   mockBatchSanitizer,
		GenericSanitizer: mockGenericSanitizer,
		Sanitizer:        mockSanitizer,
		Service:          mockService,
	}
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	controller := New(cfg)
	controller.RegisterRoutes(router)
	return router, mockService, mockSanitizer, mockGenericSanitizer, mockBatchSanitizer
}
func performLocalHostRequest(r http.Handler, method, path string, body []byte) *httptest.ResponseRecorder {
	buf := bytes.NewBuffer(body)
	req, _ := http.NewRequest(method, path, buf)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func performNonLocalRequest(r http.Handler, method, path string, body []byte) *httptest.ResponseRecorder {
	buf := bytes.NewBuffer(body)
	req, _ := http.NewRequest(method, path, buf)
	req.RemoteAddr = "1.1.1.1:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func performRequest(r http.Handler, method, path string, body []byte) *httptest.ResponseRecorder {
	buf := bytes.NewBuffer(body)
	req, _ := http.NewRequest(method, path, buf)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}
