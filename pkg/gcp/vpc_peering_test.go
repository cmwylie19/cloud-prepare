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
package gcp_test

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/cloud-prepare/pkg/api"
	"github.com/submariner-io/cloud-prepare/pkg/gcp"
)

type invalidCloud struct{}

func (f *invalidCloud) PrepareForSubmariner(input api.PrepareForSubmarinerInput, reporter api.Reporter) error {
	panic("not implemented")
}

func (f *invalidCloud) CreateVpcPeering(target api.Cloud, reporter api.Reporter) error {
	panic("not implemented")
}

func (f *invalidCloud) CleanupAfterSubmariner(reporter api.Reporter) error {
	panic("not implemented")
}

var _ = Describe("GCP Peering", func() {
	Context("VpcHelperFunctions", testVpcHelperFunctions)
	Context("CreateVpcPeering", testCreateVpcPeering)
})

func testCreateVpcPeering() {
	cloudA := newCloudTestDriver()
	When("called with a non-GCP Cloud", func() {
		It("should return an error", func() {
			invalidCloud := &invalidCloud{}
			err := cloudA.cloud.CreateVpcPeering(invalidCloud, api.NewLoggingReporter())
			Expect(err).To(HaveOccurred())
		})
	})
}

func testVpcHelperFunctions() {
	// cloudB := newTargetCloudTestDriver()

	When("GetNetworkURL is called with projectID and infraID", func() {
		It("should return short network url", func() {
			network_url := gcp.GetNetworkURL(projectID, infraID)
			Expect(network_url).To(Equal(fmt.Sprintf("projects/%s/global/networks/%s-network", projectID, infraID)))
		})
	})

	When("RemoveVpcPeeringRequest is called with infraID", func() {
		It("should return the correct NetworksRemovePeeringRequest", func() {
			removePeeringRequest := gcp.RemoveVpcPeeringRequest(infraID)
			Expect(removePeeringRequest.Name).To(Equal(fmt.Sprintf("%s-peering", infraID)))
		})
	})

	When("GeneratePeeringName is called with infraID", func() {
		It("should return the a correct peeringName", func() {
			peeringName := gcp.GeneratePeeringName(infraID)
			Expect(peeringName).To(Equal(fmt.Sprintf("%s-peering", infraID)))
		})
	})

	When("NewVpcPeeringRequest is called with targetInfraID", func() {
		It("should return the correct NetworksAddPeeringRequest", func() {
			targetNetwork := targetInfraID + "-network"
			peeringRequest := gcp.NewVpcPeeringRequest(infraID, targetNetwork)
			Expect(peeringRequest.Name).To(Equal(fmt.Sprintf("%s-peering", infraID)))
		})
	})

}

func newTargetCloudTestDriver() *cloudTestDriver {
	t := &cloudTestDriver{}

	BeforeEach(func() {
		t.beforeEach()

		t.cloud = gcp.NewCloud(gcp.CloudInfo{
			InfraID:   targetInfraID,
			Region:    targetRegion,
			ProjectID: targetProjectID,
			Client:    t.gcpClient,
		})
	})

	AfterEach(t.afterEach)

	return t
}
