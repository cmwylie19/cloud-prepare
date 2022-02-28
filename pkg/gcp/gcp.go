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
	"errors"
	"fmt"
	"strings"

	"github.com/submariner-io/cloud-prepare/pkg/api"
)

type gcpCloud struct {
	CloudInfo
}

// NewCloud creates a new api.Cloud instance which can prepare GCP for Submariner to be deployed on it.
func NewCloud(info CloudInfo) api.Cloud {
	return &gcpCloud{CloudInfo: info}
}

// PrepareForSubmariner prepares submariner cluster environment on GCP.
func (gc *gcpCloud) PrepareForSubmariner(input api.PrepareForSubmarinerInput, reporter api.Reporter) error {
	// Create the inbound firewall rule for submariner internal ports.
	reporter.Started("Opening internal ports %q for intra-cluster communications on GCP", formatPorts(input.InternalPorts))

	internalIngress := newInternalFirewallRule(gc.ProjectID, gc.InfraID, input.InternalPorts)
	if err := gc.openPorts(internalIngress); err != nil {
		reporter.Failed(err)
		return err
	}

	reporter.Succeeded("Opened internal ports %q with firewall rule %q on GCP",
		formatPorts(input.InternalPorts), internalIngress.Name)

	return nil
}

// CleanupAfterSubmariner clean up submariner cluster environment on GCP.
func (gc *gcpCloud) CleanupAfterSubmariner(reporter api.Reporter) error {
	// Delete the inbound and outbound firewall rules to close submariner internal ports.
	internalIngressName := generateRuleName(gc.InfraID, internalPortsRuleName)

	return gc.deleteFirewallRule(internalIngressName, reporter)
}

func formatPorts(ports []api.PortSpec) string {
	portStrs := []string{}
	for _, port := range ports {
		portStrs = append(portStrs, fmt.Sprintf("%d/%s", port.Port, port.Protocol))
	}

	return strings.Join(portStrs, ", ")
}

// CreateVpcPeering Creates a VPC Peering to the target cloud. Only the same
// Cloud Provider is supported.
func (gc *gcpCloud) CreateVpcPeering(target api.Cloud, reporter api.Reporter) error {
	// validate type
	_, ok := target.(*gcpCloud)
	if !ok {
		err := errors.New("only GCP clients are supported")
		reporter.Failed(err)
		return err
	}

	// https://cloud.google.com/vpc/docs/vpc-peering

	// Validate Peering does not already exist

	// get values from target
	targetProjectID, targetInfraID := ExtractValuesFromTarget(target)

	NETWORK_NAME := gc.InfraID + "-network"
	TARGET_NETWORK_NAME := targetInfraID + "-network"
	NETWORK := GetNetworkURL(gc.ProjectID, gc.InfraID)
	TARGET_NETWORK := GetNetworkURL(targetProjectID, targetInfraID)

	reporter.Started("Started VPC Peering between %q and %q", NETWORK, TARGET_NETWORK)

	// Create peering request for both networks
	peeringRequest := NewVpcPeeringRequest(gc.InfraID, TARGET_NETWORK)
	targetPeeringRequest := NewVpcPeeringRequest(targetInfraID, NETWORK)

	// Peer VPC with Target VPC (A-B)
	if err := gc.createVpcPeering(gc.ProjectID, NETWORK_NAME, peeringRequest, reporter); err != nil {
		err_msg := errors.New("Failed peering to target")
		reporter.Failed(err_msg, err)
		return err
	}

	// Peer Target VPC with VPC (B-A)
	if err := gc.createVpcPeering(targetProjectID, TARGET_NETWORK_NAME, targetPeeringRequest, reporter); err != nil {
		err_msg := errors.New("Failed peering from target to host")
		reporter.Failed(err_msg, err)
		return err
	}

	reporter.Succeeded("Peered VPCs %q and %q", NETWORK_NAME, TARGET_NETWORK_NAME)

	return nil
}

// CleanupVpcPeering Removes the VPC Peering with the target cloud and the related Routes.
func (gc *gcpCloud) CleanupVpcPeering(target api.Cloud, reporter api.Reporter) error {
	_, ok := target.(*gcpCloud)
	if !ok {
		err := errors.New("only GCP clients are supported")
		reporter.Failed(err)
		return err
	}

	// get values from target
	targetProjectID, targetInfraID := ExtractValuesFromTarget(target)

	NETWORK_NAME := gc.InfraID + "-network"
	TARGET_NETWORK_NAME := targetInfraID + "-network"
	NETWORK := GetNetworkURL(gc.ProjectID, gc.InfraID)
	TARGET_NETWORK := GetNetworkURL(targetProjectID, targetInfraID)

	reporter.Started("Started Removing VPC Peering between %q and %q", NETWORK, TARGET_NETWORK)

	// Create  remove peering request for both networks
	removePeeringRequest := RemoveVpcPeeringRequest(gc.InfraID)
	targetRemovePeeringRequest := RemoveVpcPeeringRequest(targetInfraID)

	// Peer VPC with Target VPC (A-B)
	if err := gc.deleteVpcPeering(gc.ProjectID, NETWORK_NAME, removePeeringRequest, reporter); err != nil {
		err_msg := errors.New("Failed peering removal on host")
		reporter.Failed(err, err_msg)
		return err
	}

	// Peer Target VPC with VPC (B-A)
	if err := gc.deleteVpcPeering(targetProjectID, TARGET_NETWORK_NAME, targetRemovePeeringRequest, reporter); err != nil {
		err_msg := errors.New("Failed peering removal on target")
		reporter.Failed(err, err_msg)
		return err
	}

	reporter.Succeeded("Removed Peering between VPCs %q and %q", NETWORK_NAME, TARGET_NETWORK_NAME)

	return nil

	//return errors.New("GCP CleanupVpcPeering not implemented")
}
