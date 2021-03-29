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
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth_gin"
	"github.com/gin-gonic/gin"
	"github.com/nymtech/nym/validator/nym/models"
)

const MaximumMixnodes = 1500

// explicitly declared so that a similar attack could not be used for gateways this time.
const MaximumGateways = 1000
const SystemVersion = "0.9.2"

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

	mixCount     int
	gatewayCount int

	// ensures that if two nodes register at the same time while there's only a single opening left, they won't
	// both be accepted.
	// note that the lock is not used when creating mix status or unregistering nodes as those can only DECREMENT
	// mix count
	registrationLock sync.Mutex
}

// Controller ...
type Controller interface {
	CreateMixStatus(c *gin.Context)
	RegisterRoutes(router *gin.Engine)
}

// New returns a new mixmining.Controller
func New(cfg Config) Controller {
	initialMixCount := cfg.Service.MixCount()
	initialGatewayCount := cfg.Service.GatewayCount()

	// move all non 0.9.2 nodes to "removed" set
	cfg.Service.StartupPurge()

	return &controller{cfg.Service, cfg.Sanitizer, cfg.GenericSanitizer, cfg.BatchSanitizer, initialMixCount, initialGatewayCount, sync.Mutex{}}
}

func (controller *controller) RegisterRoutes(router *gin.Engine) {
	// use that limiter if no other is specified (1 request per second)
	lmt := tollbooth_gin.LimitHandler(tollbooth.NewLimiter(1, nil))

	// allows 2 requests per minute (meaning in a 1min you can register and unregister)
	registrationLmt := tollbooth_gin.LimitHandler(tollbooth.NewLimiter(2.0/60.0, nil).SetBurst(2))

	// allows 1 topology request per 10s
	topologyLmt := tollbooth_gin.LimitHandler(tollbooth.NewLimiter(1.0/10.0, nil).SetBurst(3))

	router.POST("/api/mixmining", lmt, controller.CreateMixStatus)
	router.POST("/api/mixmining/batch", lmt, controller.BatchCreateMixStatus)
	router.GET("/api/mixmining/node/:pubkey/history", lmt, controller.ListMeasurements)
	router.GET("/api/mixmining/node/:pubkey/report", lmt, controller.GetMixStatusReport)
	router.GET("/api/mixmining/fullreport", lmt, controller.BatchGetMixStatusReport)

	router.POST("/api/mixmining/register/mix", registrationLmt, controller.RegisterMixPresence)
	router.POST("/api/mixmining/register/gateway", registrationLmt, controller.RegisterGatewayPresence)
	router.DELETE("/api/mixmining/register/:id", registrationLmt, controller.UnregisterPresence)
	router.GET("/api/mixmining/topology", topologyLmt, controller.GetTopology)
	router.GET("/api/mixmining/topology/active", topologyLmt, controller.GetActiveTopology)
	router.PATCH("/api/mixmining/reputation/:id", lmt, controller.ChangeReputation)

	router.GET("/api/mixmining/topology/removed", topologyLmt, controller.GetRemovedTopology)
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

	// we don't know how number of active nodes changed - update it
	controller.mixCount = controller.service.MixCount()
	controller.gatewayCount = controller.service.GatewayCount()

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

	// we don't know how number of active nodes changed - update it
	controller.mixCount = controller.service.MixCount()
	controller.gatewayCount = controller.service.GatewayCount()

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

// RegisterMixPresence ...
// @Summary Lets a mixnode tell the directory server it's coming online
// @Description On Nym nodes startup they register their presence indicating they should be alive and get added to the set of active nodes in the topology.
// @ID registerMixPresence
// @Accept  json
// @Produce  json
// @Tags mixmining
// @Param   object      body   models.MixRegistrationInfo     true  "object"
// @Success 200
// @Failure 400 {object} models.Error
// @Failure 404 {object} models.Error
// @Failure 409 {object} models.Error
// @Failure 500 {object} models.Error
// @Router /api/mixmining/register/mix [post]
func (controller *controller) RegisterMixPresence(ctx *gin.Context) {
	if controller.mixCount >= MaximumMixnodes {
		ctx.JSON(http.StatusConflict, gin.H{"error": "mixnet is already at capacity"})
		return
	}

	var presence models.MixRegistrationInfo
	if err := ctx.ShouldBindJSON(&presence); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	controller.genericSanitizer.Sanitize(&presence)

	if controller.service.CheckForDuplicateIP(presence.MixHost) {
		ctx.JSON(http.StatusConflict, gin.H{"error": "node with the same ip address already exists"})
		return
	}

	if presence.Version != SystemVersion {
		ctx.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("running non %v version", SystemVersion)})
		return
	}

	// make sure there's still space left after all the checks
	// (another node might have finished registration in the meantime)
	// it is done before locking as it's a very cheap check
	if controller.mixCount >= MaximumMixnodes {
		ctx.JSON(http.StatusConflict, gin.H{"error": "mixnet is already at capacity"})
		return
	}

	controller.registrationLock.Lock()
	defer controller.registrationLock.Unlock()

	// perform the same check after locking because we don't know how long it took to acquire the lock
	if controller.mixCount >= MaximumMixnodes {
		ctx.JSON(http.StatusConflict, gin.H{"error": "mixnet is already at capacity"})
		return
	}

	controller.service.RegisterMix(presence)
	controller.mixCount = controller.service.MixCount()

	ctx.JSON(http.StatusOK, gin.H{"ok": true})
}

// RegisterGatewayPresence ...
// @Summary Lets a gateway tell the directory server it's coming online
// @Description On Nym nodes startup they register their presence indicating they should be alive and get added to the set of active nodes in the topology.
// @ID registerGatewayPresence
// @Accept  json
// @Produce  json
// @Tags mixmining
// @Param   object      body   models.GatewayRegistrationInfo     true  "object"
// @Success 200
// @Failure 400 {object} models.Error
// @Failure 404 {object} models.Error
// @Failure 409 {object} models.Error
// @Failure 500 {object} models.Error
// @Router /api/mixmining/register/gateway [post]
func (controller *controller) RegisterGatewayPresence(ctx *gin.Context) {
	if controller.gatewayCount >= MaximumGateways {
		ctx.JSON(http.StatusConflict, gin.H{"error": "mixnet is already at capacity"})
		return
	}

	var presence models.GatewayRegistrationInfo
	if err := ctx.ShouldBindJSON(&presence); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	controller.genericSanitizer.Sanitize(&presence)

	if controller.service.CheckForDuplicateIP(presence.MixHost) {
		ctx.JSON(http.StatusConflict, gin.H{"error": "gateway with the same ip address already exists"})
		return
	}

	if presence.Version != SystemVersion {
		ctx.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("running non %v version", SystemVersion)})
		return
	}

	controller.service.RegisterGateway(presence)
	controller.gatewayCount = controller.service.GatewayCount()

	ctx.JSON(http.StatusOK, gin.H{"ok": true})
}

// UnregisterPresence ...
// @Summary Unregister presence of node.
// @Description Messages sent by a node on powering down to indicate it's going offline so that it should get removed from active topology.
// @ID unregisterPresence
// @Accept  json
// @Produce  json
// @Tags mixmining
// @Param id path string true "Node Identity"
// @Success 200
// @Failure 400 {object} models.Error
// @Failure 403 {object} models.Error
// @Failure 404 {object} models.Error
// @Failure 500 {object} models.Error
// @Router /api/mixmining/register/{id} [delete]
func (controller *controller) UnregisterPresence(ctx *gin.Context) {
	remoteIP := ctx.ClientIP()

	id := ctx.Param("id")
	controller.genericSanitizer.Sanitize(&id)

	status, err := controller.service.UnregisterNode(id, remoteIP)
	if err != nil {
		ctx.JSON(status, gin.H{"error": err.Error()})
	} else {
		// status MUST be 200, otherwise service is really messed up
		if status != http.StatusOK {
			fmt.Fprintf(os.Stderr, "unregistration did not return an error, but the status was not 200!\n")
		}
		controller.mixCount = controller.service.MixCount()
		controller.gatewayCount = controller.service.GatewayCount()
		ctx.JSON(status, gin.H{"ok": true})
	}
}

// GetTopology ...
// @Summary Lists Nym mixnodes and gateways on the network alongside their reputation.
// @Description On Nym nodes startup they register their presence indicating they should be alive. This method provides a list of nodes which have done so.
// @ID getTopology
// @Produce  json
// @Tags mixmining
// @Success 200 {object} models.Topology
// @Failure 500 {object} models.Error
// @Router /api/mixmining/topology [get]
func (controller *controller) GetTopology(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, controller.service.GetTopology())
}

// GetActiveTopology ...
// @Summary Lists Nym mixnodes and gateways on the network alongside their reputation, such that the reputation is at least 100.
// @Description On Nym nodes startup they register their presence indicating they should be alive. This method provides a list of nodes which have done so.
// @ID getActiveTopology
// @Produce  json
// @Tags mixmining
// @Success 200 {object} models.Topology
// @Failure 500 {object} models.Error
// @Router /api/mixmining/topology/active [get]
func (controller *controller) GetActiveTopology(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, controller.service.GetActiveTopology())
}

// ChangeReputation ...
// @Summary Change reputation of a node
// @Description Changes reputation of given node to some specified value
// @ID changeReputation
// @Accept  json
// @Produce  json
// @Tags mixmining
// @Param id path string true "Node Identity"
// @Param reputation query integer true "New Reputation"
// @Success 200
// @Failure 400 {object} models.Error
// @Failure 403 {object} models.Error
// @Failure 404 {object} models.Error
// @Failure 500 {object} models.Error
// @Router /api/mixmining/reputation/{id} [patch]
// NOTE: it's only accessible from localhost and its only purpose is to jumpstart the network quickly (so you could
// manually set few nodes above threshold reputation rather than to wait for enough reports to come in)
func (controller *controller) ChangeReputation(ctx *gin.Context) {
	remoteIP := ctx.ClientIP()
	if !(remoteIP == "127.0.0.1" || remoteIP == "::1" || ctx.Request.RemoteAddr == "127.0.0.1" || ctx.Request.RemoteAddr == "::1") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	id := ctx.Param("id")
	newRepStr := ctx.Request.URL.Query().Get("reputation")
	controller.genericSanitizer.Sanitize(&id)
	controller.genericSanitizer.Sanitize(&newRepStr)

	newRep, err := strconv.Atoi(newRepStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if controller.service.SetReputation(id, int64(newRep)) {
		ctx.JSON(http.StatusOK, gin.H{"ok": true})
	} else {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "entry does not exist"})
	}
}

// GetRemovedTopology ...
// @Summary Lists Nym mixnodes and gateways on the network that got removed due to bad service provided.
// @Description On Nym nodes startup they register their presence indicating they should be alive.
// This method provides a list of nodes which have done so but failed to provide good quality service.
// @ID getRemovedTopology
// @Produce  json
// @Tags mixmining
// @Success 200 {object} models.Topology
// @Failure 500 {object} models.Error
// @Router /api/mixmining/topology/removed [get]
func (controller *controller) GetRemovedTopology(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, controller.service.GetRemovedTopology())
}
