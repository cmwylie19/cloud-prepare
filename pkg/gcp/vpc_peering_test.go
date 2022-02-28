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
	// "errors"
	// "fmt"
	// "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	// . "github.com/onsi/gomega"
	// "github.com/submariner-io/cloud-prepare/pkg/api"
	"github.com/submariner-io/cloud-prepare/pkg/gcp"
	// "google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
	"net/http"
)

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

const peeringName = "test-infraID-peering"
const networkName = "test-infraID-network"

// const targetPeeringName = "test-target-infraID-peering"

var _ = Describe("Cloud", func() {
	Describe("CreateVpcPeering", testCreateVpcPeering)
	// Describe("CleanupVpcPeering", testCleanupVpcPeering)
})

func testCreateVpcPeering() {
	t := newCloudTestDriver()
	// target := newTargetCloudTestDriver()

	// var retError error

	// JustBeforeEach(func() {
	// 	retError = t.cloud.PrepareForSubmariner(api.PrepareForSubmarinerInput{
	// 		InternalPorts: []api.PortSpec{
	// 			{
	// 				Port:     100,
	// 				Protocol: "TCP",
	// 			},
	// 			{
	// 				Port:     200,
	// 				Protocol: "UDP",
	// 			},
	// 		},
	// 	}, api.NewLoggingReporter())
	// })

	When("the vpc peering doesn't exist", func() {
		BeforeEach(func() {
			t.gcpClient.EXPECT().GetVpcPeering(projectID, peeringName).Return(nil, &googleapi.Error{Code: http.StatusNotFound})
		})

		// Context("", func() {
		// 	var actualRequest *compute.NetworksAddPeeringRequest
		// 	// var actualRule *compute.Firewall

		// 	BeforeEach(func() {
		// 		t.gcpClient.EXPECT().PeerVpcs(projectID, networkName, gomock.Any()).DoAndReturn(func(_ string, _ string, peerRequest *compute.NetworksAddPeeringRequest) error {
		// 			actualRequest = peerRequest
		// 			return nil
		// 		})
		// 	})

		// 	It("should correctly peer it", func() {
		// 		// Expect(retError).To(Succeed())
		// 		fmt.Printf("ActualRequest: %+v", actualRequest)
		// 		Expect(actualRequest).ToNot(BeNil(), "PeerVpcs was not called")
		// 		// assertIngressRule(actualNetwork)
		// 	})
		// })

		// Context("and insertion of other peering fails", func() {
		// 	BeforeEach(func() {
		// 		t.gcpClient.EXPECT().PeerVpcs(projectID, networkName, gomock.Any()).Return(errors.New("fake insert error"))
		// 	})

		// 	It("should return an error", func() {
		// 		Expect(retError).ToNot(Succeed())
		// 	})
		// })
	})

	// When("the peering already exists", func() {
	// 	BeforeEach(func() {
	// 		t.gcpClient.EXPECT().GetFirewallRule(projectID, ingressRuleName).DoAndReturn(func(_, ruleName string) (*compute.Firewall, error) {
	// 			return &compute.Firewall{Name: ruleName}, nil
	// 		})
	// 	})

	// 	Context("", func() {
	// 		var actualRule *compute.Firewall

	// 		BeforeEach(func() {
	// 			t.gcpClient.EXPECT().UpdateFirewallRule(projectID, ingressRuleName, gomock.Any()).DoAndReturn(
	// 				func(_, _ string, rule *compute.Firewall) error {
	// 					actualRule = rule
	// 					return nil
	// 				})
	// 		})

	// 		It("should update it", func() {
	// 			Expect(retError).To(Succeed())

	// 			Expect(actualRule).ToNot(BeNil(), "UpdateFirewallRule was not called")
	// 			assertIngressRule(actualRule)
	// 		})
	// 	})

	// 	Context("and update fails", func() {
	// 		BeforeEach(func() {
	// 			t.gcpClient.EXPECT().UpdateFirewallRule(projectID, ingressRuleName, gomock.Any()).Return(errors.New("fake update error"))
	// 		})

	// 		It("should return an error", func() {
	// 			Expect(retError).ToNot(Succeed())
	// 		})
	// 	})
	// })

	// When("retrieval of the firewall rule fails", func() {
	// 	BeforeEach(func() {
	// 		t.gcpClient.EXPECT().GetFirewallRule(projectID, ingressRuleName).Return(nil, errors.New("fake get error"))
	// 	})

	// 	It("should return an error", func() {
	// 		Expect(retError).ToNot(Succeed())
	// 	})
	// })
}

// func testCleanupVpcPeering() {
// 	t := newCloudTestDriver()

// 	var retError error

// 	JustBeforeEach(func() {
// 		retError = t.cloud.CleanupAfterSubmariner(api.NewLoggingReporter())
// 	})

// 	Context("on success", func() {
// 		BeforeEach(func() {
// 			t.gcpClient.EXPECT().DeleteFirewallRule(projectID, ingressRuleName).Return(nil)
// 		})

// 		It("should delete the firewall rule", func() {
// 			Expect(retError).To(Succeed())
// 		})
// 	})

// 	When("the peering doesn't exist", func() {
// 		BeforeEach(func() {
// 			t.gcpClient.EXPECT().DeleteFirewallRule(projectID, ingressRuleName).Return(&googleapi.Error{Code: http.StatusNotFound})
// 		})

// 		It("should succeed", func() {
// 			Expect(retError).To(Succeed())
// 		})
// 	})

// 	When("deletion fails", func() {
// 		BeforeEach(func() {
// 			t.gcpClient.EXPECT().DeleteFirewallRule(projectID, ingressRuleName).Return(errors.New("fake delete error"))
// 		})

// 		It("should return an error", func() {
// 			Expect(retError).ToNot(Succeed())
// 		})
// 	})
// }

// func assertIngressRule(rule *compute.Firewall) {
// 	Expect(rule.Name).To(Equal(ingressRuleName))
// 	Expect(rule.Direction).To(Equal("INGRESS"))
// 	Expect(rule.Allowed).To(HaveLen(2))
// 	Expect(rule.Allowed[0]).To(Equal(&compute.FirewallAllowed{
// 		IPProtocol: "TCP",
// 		Ports:      []string{"100"},
// 	}))
// 	Expect(rule.Allowed[1]).To(Equal(&compute.FirewallAllowed{
// 		IPProtocol: "UDP",
// 		Ports:      []string{"200"},
// 	}))
// }
