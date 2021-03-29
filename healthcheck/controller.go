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

package healthcheck

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth_gin"
)

// controller is the presence controller
type controller struct{}

// Controller is the presence controller
type Controller interface {
	HealthCheck(c *gin.Context)
	RegisterRoutes(router *gin.Engine)
}

// New returns a new pki.Controller
func New() Controller {
	return &controller{}
}

func (controller *controller) RegisterRoutes(router *gin.Engine) {
	router.GET("/api/healthcheck", tollbooth_gin.LimitHandler(tollbooth.NewLimiter(1, nil)), controller.HealthCheck)
}

// HealthCheck ...
// @Summary Lets the directory server tell the world it's alive.
// @Description Returns a 200 if the directory server is available. Good route to use for automated monitoring.
// @ID healthCheck
// @Accept  json
// @Produce  json
// @Tags healthcheck
// @Success 200
// @Router /api/healthcheck [get]
func (controller *controller) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
