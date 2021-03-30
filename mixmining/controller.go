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
	"net/http"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth_gin"
	"github.com/gin-gonic/gin"
	"github.com/nymtech/node-status-api/models"
)

// Config for this controller
type Config struct {
	BatchSanitizer   BatchSanitizer   // batch mix reports
	GenericSanitizer GenericSanitizer // originally introduced for what was in mix registration
	Sanitizer        Sanitizer        // mix reports
	Service          IService
}

// controller is the mixmining controller
type controller struct {
	service          IService
	sanitizer        Sanitizer
	genericSanitizer GenericSanitizer
	batchSanitizer   BatchSanitizer
}

// Controller ...
type Controller interface {
	CreateMixStatus(c *gin.Context)
	RegisterRoutes(router *gin.Engine)
}

// New returns a new mixmining.Controller
func New(cfg Config) Controller {
	return &controller{cfg.Service, cfg.Sanitizer, cfg.GenericSanitizer, cfg.BatchSanitizer}
}

func (controller *controller) RegisterRoutes(router *gin.Engine) {
	// use that limiter if no other is specified (1 request per second)
	lmt := tollbooth_gin.LimitHandler(tollbooth.NewLimiter(1, nil))

	router.POST("/api/mixmining", lmt, controller.CreateMixStatus)
	router.POST("/api/mixmining/batch", lmt, controller.BatchCreateMixStatus)
	router.GET("/api/mixmining/node/:pubkey/history", lmt, controller.ListMeasurements)
	router.GET("/api/mixmining/node/:pubkey/report", lmt, controller.GetMixStatusReport)
	router.GET("/api/mixmining/fullreport", lmt, controller.BatchGetMixStatusReport)
}

// ListMeasurements lists mixnode statuses
// @Summary Lists mixnode activity
// @Description Lists all mixnode statuses for a given node pubkey
// @ID listMixStatuses
// @Accept  json
// @Produce  json
// @Tags mixmining
// @Param pubkey path string true "Mixnode Pubkey"
// @Success 200 {array} models.MixStatus
// @Failure 400 {object} models.Error
// @Failure 404 {object} models.Error
// @Failure 500 {object} models.Error
// @Router /api/mixmining/node/{pubkey}/history [get]
func (controller *controller) ListMeasurements(c *gin.Context) {
	pubkey := c.Param("pubkey")
	measurements := controller.service.ListMixStatus(pubkey)
	c.JSON(http.StatusOK, measurements)
}

// CreateMixStatus ...
// @Summary Lets the network monitor create a new uptime status for a mix
// @Description Nym network monitor sends packets through the system and checks if they make it. The network monitor then hits this method to report whether the node was up at a given time.
// @ID addMixStatus
// @Accept  json
// @Produce  json
// @Tags mixmining
// @Param   object      body   models.MixStatus     true  "object"
// @Success 201
// @Failure 400 {object} models.Error
// @Failure 403 {object} models.Error
// @Failure 404 {object} models.Error
// @Failure 500 {object} models.Error
// @Router /api/mixmining [post]
func (controller *controller) CreateMixStatus(c *gin.Context) {
	remoteIP := c.ClientIP()
	if !(remoteIP == "127.0.0.1" || remoteIP == "::1" || c.Request.RemoteAddr == "127.0.0.1" || c.Request.RemoteAddr == "::1") {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	var status models.MixStatus
	if err := c.ShouldBindJSON(&status); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sanitized := controller.sanitizer.Sanitize(status)
	persisted := controller.service.CreateMixStatus(sanitized)
	controller.service.SaveStatusReport(persisted)

	c.JSON(http.StatusCreated, gin.H{"ok": true})
}

// GetMixStatusReport ...
// @Summary Retrieves a summary report of historical mix status
// @Description Provides summary uptime statistics for last 5 minutes, day, week, and month
// @ID getMixStatusReport
// @Accept  json
// @Produce  json
// @Tags mixmining
// @Param pubkey path string true "Mixnode Pubkey"
// @Success 200
// @Failure 400 {object} models.Error
// @Failure 404 {object} models.Error
// @Failure 500 {object} models.Error
// @Router /api/mixmining/node/{pubkey}/report [get]
func (controller *controller) GetMixStatusReport(c *gin.Context) {
	pubkey := c.Param("pubkey")
	report := controller.service.GetStatusReport(pubkey)
	if (report == models.MixStatusReport{}) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	}
	c.JSON(http.StatusOK, report)
}

// BatchCreateMixStatus ...
// @Summary Lets the network monitor create a new uptime status for multiple mixes
// @Description Nym network monitor sends packets through the system and checks if they make it. The network monitor then hits this method to report whether nodes were up at a given time.
// @ID batchCreateMixStatus
// @Accept  json
// @Produce  json
// @Tags mixmining
// @Param   object      body   models.BatchMixStatus     true  "object"
// @Success 201
// @Failure 400 {object} models.Error
// @Failure 403 {object} models.Error
// @Failure 404 {object} models.Error
// @Failure 500 {object} models.Error
// @Router /api/mixmining/batch [post]
func (controller *controller) BatchCreateMixStatus(c *gin.Context) {
	remoteIP := c.ClientIP()
	if !(remoteIP == "127.0.0.1" || remoteIP == "::1" || c.Request.RemoteAddr == "127.0.0.1" || c.Request.RemoteAddr == "::1") {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	var status models.BatchMixStatus
	if err := c.ShouldBindJSON(&status); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sanitized := controller.batchSanitizer.Sanitize(status)

	persisted := controller.service.BatchCreateMixStatus(sanitized)
	controller.service.SaveBatchStatusReport(persisted)

	c.JSON(http.StatusCreated, gin.H{"ok": true})
}

// BatchGetMixStatusReport ...
// @Summary Retrieves a summary report of historical mix status
// @Description Provides summary uptime statistics for last 5 minutes, day, week, and month
// @ID batchGetMixStatusReport
// @Accept  json
// @Produce  json
// @Tags mixmining
// @Success 200
// @Failure 400 {object} models.Error
// @Failure 404 {object} models.Error
// @Failure 500 {object} models.Error
// @Router /api/mixmining/fullreport [get]
func (controller *controller) BatchGetMixStatusReport(c *gin.Context) {
	report := controller.service.BatchGetMixStatusReport()
	c.JSON(http.StatusOK, report)
}
