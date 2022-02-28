/*
SPDX-License-Identifier: Apache-2.0

Copyright Contributors to the Submariner project.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package gcp

import (
	"fmt"
	"reflect"

	"github.com/submariner-io/cloud-prepare/pkg/api"
	"google.golang.org/api/compute/v1"
)

func RemoveVpcPeeringRequest(infraID string) *compute.NetworksRemovePeeringRequest {
	return &compute.NetworksRemovePeeringRequest{
		Name: GeneratePeeringName(infraID),
	}
}
func NewVpcPeeringRequest(infraID, targetNetwork string) *compute.NetworksAddPeeringRequest {
	return &compute.NetworksAddPeeringRequest{
		Name:             GeneratePeeringName(infraID),
		PeerNetwork:      targetNetwork,
		AutoCreateRoutes: true,
		// This causes the request to fail, leave commented for review

		// NetworkPeering: &compute.NetworkPeering{
		// 	ImportCustomRoutes:   true,
		// 	ExchangeSubnetRoutes: true,
		// },
	}
}

func GeneratePeeringName(infraID string) string {
	return fmt.Sprintf("%s-peering", infraID)
}

// Format network short URL
func GetNetworkURL(projectID, infraID string) string {
	return fmt.Sprintf("projects/%s/global/networks/%s-network", projectID, infraID)
}

// Extract values from target
func ExtractValuesFromTarget(target api.Cloud) (string, string) {
	// Extract CloudInfo from Target
	target_values := reflect.ValueOf(target).Elem()
	cloud_info := target_values.FieldByName("CloudInfo")

	projectID := fmt.Sprintf("%s", cloud_info.FieldByName("ProjectID"))
	infraID := fmt.Sprintf("%s", cloud_info.FieldByName("InfraID"))

	return projectID, infraID
}
