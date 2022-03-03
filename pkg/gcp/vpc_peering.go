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
	"time"

	"google.golang.org/api/compute/v1"
)

const (
	attempts = 3
	waitTime = 10
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

func RunWithRetries(numSeconds int, f func() error) error {
	var err error
	for retries := attempts; retries > 0; {
		err = f()
		if err != nil {
			retries--

			time.Sleep(time.Duration(numSeconds) * time.Second)
		} else {
			return nil
		}
	}

	return err
}
