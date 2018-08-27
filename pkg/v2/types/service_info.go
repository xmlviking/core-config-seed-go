/*******************************************************************************
 * Copyright 2018 Dell Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under the License
 * is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express
 * or implied. See the License for the specific language governing permissions and limitations under
 * the License.
 *******************************************************************************/
package types

// ServiceInfo contains configuration settings necessary for the basic operation of any EdgeX service.
type ServiceInfo struct {
	// Host is the hostname or IP address of the service.
	Host           string
	// Port is the HTTP port of the service.
	Port           int
	// The protocol that should be used to call this service
	Protocol       string
	// HealthCheck is a URL specifying a healthcheck REST
	// endpoint used by the Registry to determine if the
	// service is available.
	HealthCheck    string
	// Health check interval
	CheckInterval string
	// StartupMsg specifies a string to log once service
	// initialization and startup is completed.
	StartupMsg     string
	// ReadMaxLimit specifies the maximum size list supported
	// in response to REST calls to other services.
	ReadMaxLimit   int
	// Timeout specifies a timeout (in milliseconds) for
	// processing REST calls from other services.
	Timeout        int
}