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

// MetaData : Struct used to parse the JSON configuration file
// Thought here is for MetaData services we only have a Host and Port
// The services should be able to dynamically obtain the required information based on a prior knowledge
// This is a temporary structure to be removed in the future.
type MetaDataInfo struct {
	ProvisionWatcherURL  string
	ProvisionWatcherPath string
	DevicePath string
	DeviceURL string
	CommandPath string
	CommandURL string
}
