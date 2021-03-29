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

package main

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/microcosm-cc/bluemonday"
	"github.com/nymtech/node-status-api/mixmining"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	directory := New()
	directory.Run(":8081")
}

// New returns a new node status REST API server
// @title Nym Node Status API
// @version 0.10.0-dev
// @description A node status API that holds uptime information for Nym nodes.
// @termsOfService http://swagger.io/terms/
// @license.name Apache 2.0
// @license.url https://github.com/nymtech/node-status-api/license
func New() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	// Set the router as the default one shipped with Gin
	router := gin.Default()

	// Add cors middleware
	router.Use(cors.Default())

	// Serve Swagger frontend static files using gin-swagger middleware
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Sanitize controller input against XSS attacks using bluemonday.Policy
	policy := bluemonday.UGCPolicy()

	// Measurements: wire up dependency injection
	measurementsCfg := injectMeasurements(policy)

	// Register all HTTP controller routes
	mixmining.New(measurementsCfg).RegisterRoutes(router)

	return router
}

func injectMeasurements(policy *bluemonday.Policy) mixmining.Config {
	sanitizer := mixmining.NewSanitizer(policy)
	batchSanitizer := mixmining.NewBatchSanitizer(policy)
	genericSanitizer := mixmining.NewGenericSanitizer(policy)
	db := mixmining.NewDb(false)
	mixminingService := *mixmining.NewService(db, false)

	return mixmining.Config{
		Service:          &mixminingService,
		Sanitizer:        sanitizer,
		GenericSanitizer: genericSanitizer,
		BatchSanitizer:   batchSanitizer,
	}
}
