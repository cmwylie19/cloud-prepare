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
package azure

import (
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/submariner-io/cloud-prepare/pkg/api"
)

const (
	internalSecurityGroupSuffix = "-nsg"
	externalSecurityGroupSuffix = "-submariner-external-sg"
	internalSecurityRulePrefix  = "Submariner-Internal-"
	externalSecurityRulePrefix  = "Submariner-External-"
	inboundRulePrefix           = "Submariner-Inbound-"
	frontendIPConfigurationName = "public-lb-ip-v4"
	allNetworkCIDR              = "0.0.0.0/0"
	basePriorityInternal        = 2500
	baseExternalInternal        = 3500
)

type azureCloud struct {
	CloudInfo
}

// NewCloud creates a new api.Cloud instance which can prepare RHOS for Submariner to be deployed on it.
func NewCloud(info *CloudInfo) api.Cloud {
	return &azureCloud{
		CloudInfo: *info,
	}
}

func (az *azureCloud) PrepareForSubmariner(input api.PrepareForSubmarinerInput, reporter api.Reporter) error {
	reporter.Started("Opening internal ports for intra-cluster communications on Azure")

	nsgClient := getNsgClient(az.CloudInfo.SubscriptionID, az.CloudInfo.Authorizer)

	if err := az.openInternalPorts(az.InfraID, input.InternalPorts, nsgClient); err != nil {
		reporter.Failed(err)
		return err
	}

	reporter.Succeeded("Opened internal ports %q for intra-cluster communications on Azure",
		formatPorts(input.InternalPorts))

	return nil
}

func (az *azureCloud) CleanupAfterSubmariner(reporter api.Reporter) error {
	reporter.Started("Revoking intra-cluster communication permissions")

	nsgClient := getNsgClient(az.CloudInfo.SubscriptionID, az.CloudInfo.Authorizer)

	if err := az.removeInternalFirewallRules(az.InfraID, nsgClient); err != nil {
		reporter.Failed(err)
		return err
	}

	reporter.Succeeded("Revoked intra-cluster communication permissions")

	return nil
}

func getNsgClient(subscriptionID string, authorizer autorest.Authorizer) *network.SecurityGroupsClient {
	nsgClient := network.NewSecurityGroupsClient(subscriptionID)
	nsgClient.Authorizer = authorizer

	return &nsgClient
}

func formatPorts(ports []api.PortSpec) string {
	portStrs := []string{}
	for _, port := range ports {
		portStrs = append(portStrs, fmt.Sprintf("%d/%s", port.Port, port.Protocol))
	}

	return strings.Join(portStrs, ", ")
}
