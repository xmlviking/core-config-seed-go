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

// TODO: Cannot currently use primitive types in the mapping of struct to map[string]interface{}
// type DatabaseType int

const (
	// TODO: Cannot currently use primvitive subtypes
	// Mongo DatabaseType = iota

	// temp until we figure the parsing
	Mongo = iota
)

// DatabaseInfo defines the parameters necessary for connecting to the desired persistence layer.
type DatabaseInfo struct {
	Type           string
	Timeout        int
	Host           string
	Port           int
	Username       string
	Password       string
	Name           string
}