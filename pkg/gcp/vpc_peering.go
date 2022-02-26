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

	"google.golang.org/api/compute/v1"
)

func newVpcPeeringRequest(infraID, targetNetwork string) *compute.NetworksAddPeeringRequest {
	return &compute.NetworksAddPeeringRequest{
		Name:        generatePeeringName(infraID),
		PeerNetwork: targetNetwork,
		NetworkPeering: &compute.NetworkPeering{
			ImportCustomRoutes:   true,
			Network: targetNetwork,
			ExchangeSubnetRoutes: true,
		},
	}
}

func generatePeeringName(infraID string) string {
	return fmt.Sprintf("%s-peering", infraID)
}

// // update to accommodate custom ports
// func getVpcPeeringPorts() []api.PortSpec {
// 	return []api.PortSpec{
// 		{
// 			Port:     500,
// 			Protocol: "UDP",
// 		},
// 		{
// 			Port:     4500,
// 			Protocol: "UDP",
// 		},
// 		{
// 			Port:     4800,
// 			Protocol: "UDP",
// 		},
// 	}
// }
