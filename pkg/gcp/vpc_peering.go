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

func newVpcPeeringRequest(network, targetNetwork string) (peeringRequest *compute.NetworksAddPeeringRequest) {
	return &compute.NetworksAddPeeringRequest{
		Name:        generatePeeringName(network, targetNetwork),
		PeerNetwork: targetNetwork,
		NetworkPeering: &compute.NetworkPeering{
			ImportCustomRoutes:   true,
			ExchangeSubnetRoutes: true,
		},
	}
}

func generatePeeringName(network, targetNetwork string) (peeringName string) {
	return fmt.Sprintf("%s-%s-peering", network, targetNetwork)
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
