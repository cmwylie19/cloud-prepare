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
package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/submariner-io/cloud-prepare/pkg/api"
)

var _ = Describe("AWS Peering", func() {
	Context("Test Create AWS Peering", testCreateAWSPeering)
	Context("Test Get Route Table ID", testGetRouteTableID)
	Context("Test Request Peering", testRequestPeering)
	Context("Test Accept Peering", testAcceptPeering)
	Context("Test Create Routes for Peering", testCreateRoutesForPeering)
	Context("Test Clean up VPC Peering", testCleanupVpcPeering)
	Context("Test Delete VPC Peering Routes", testDeleteVpcPeeringRoutes)
})

func testCreateAWSPeering() {
	cloudA := newCloudTestDriver(infraID, region)
	cloudB := newCloudTestDriver(targetInfraID, targetRegion)

	When("prerequisites are not met", func() {
		It("receives an overlapping CidrBlock for source and target", func() {
			cloudA.awsClient.EXPECT().
				DescribeVpcs(context.TODO(), gomock.Any()).
				Return(getVpcOutputFor("vpc-a", "10.0.0.0/12"), nil)

			cloudB.awsClient.EXPECT().
				DescribeVpcs(context.TODO(), gomock.Any()).
				Return(getVpcOutputFor("vpc-b", "10.1.0.0/12"), nil)
			err := cloudA.cloud.CreateVpcPeering(cloudB.cloud, api.NewLoggingReporter())
			Expect(err).To(HaveOccurred())
			Expect(err).Should(
				MatchError("unable to validate vpc peering prerequisites: source [10.0.0.0/12] and target [10.1.0.0/12] CIDR Blocks must not overlap"),
			)
		})
	})
	When("retrieving the VPC IDs", func() {
		It("fails to get the source VPC ID", func() {
			mockDescribeVPCs(cloudA, cloudB)
			cloudA.awsClient.EXPECT().DescribeVpcs(gomock.Any(), gomock.Any()).
				Return(nil, errors.Errorf("some error"))
			err := cloudA.cloud.CreateVpcPeering(cloudB.cloud, api.NewLoggingReporter())

			Expect(err).To(HaveOccurred())
			Expect(err).Should(MatchError(MatchRegexp("unable to retrieve source VPC ID")))
		})
		It("fails to get the target VPC ID", func() {
			mockDescribeVPCs(cloudA, cloudB)
			cloudA.awsClient.EXPECT().
				DescribeVpcs(context.TODO(), gomock.Any()).
				Return(getVpcOutputFor("vpc-a", "10.0.0.0/12"), nil)
			cloudB.awsClient.EXPECT().DescribeVpcs(gomock.Any(), gomock.Any()).
				Return(nil, errors.Errorf("some error"))
			err := cloudA.cloud.CreateVpcPeering(cloudB.cloud, api.NewLoggingReporter())

			Expect(err).To(HaveOccurred())
			Expect(err).Should(MatchError(MatchRegexp("unable to retrieve target VPC ID")))
		})
	})
	When("fails to request the VPC peering", func() {
		It("fails to create the vpc peering connection", func() {
			mockDescribeVPCs(cloudA, cloudB)
			mockDescribeVPCs(cloudA, cloudB)
			cloudA.awsClient.EXPECT().CreateVpcPeeringConnection(context.TODO(), HasCreateVpcPeeringConnectionInput(
				"vpc-a",
				"vpc-b",
				"other-region",
				"test-infraID-other-infraID",
			)).Return(nil, errors.Errorf("cannot create a vpc peering connection"))

			err := cloudA.cloud.CreateVpcPeering(cloudB.cloud, api.NewLoggingReporter())

			Expect(err).To(HaveOccurred())
			Expect(err).Should(MatchError(MatchRegexp("unable to request VPC peering")))
		})
		It("fails to accept the VPC peering after exhausting the retries", func() {
			mockDescribeVPCs(cloudA, cloudB)
			mockDescribeVPCs(cloudA, cloudB)
			mockRequestPeering(cloudA)

			cloudB.awsClient.EXPECT().AcceptVpcPeeringConnection(context.TODO(), gomock.Any()).
				Return(nil, errors.New("I will not create the VPC peering")).Times(3)

			err := cloudA.cloud.CreateVpcPeering(cloudB.cloud, api.NewLoggingReporter())
			Expect(err).To(HaveOccurred())
		})
		It("fails to create the routes for the VPC peering after exhausting the retries", func() {
			mockDescribeVPCs(cloudA, cloudB)
			mockDescribeVPCs(cloudA, cloudB)
			mockRequestPeering(cloudA)
			mockAcceptPeering(cloudB)

			cloudA.awsClient.EXPECT().DescribeRouteTables(context.TODO(), gomock.Any()).
				Return(nil, errors.New("I will not describe the RouteTables")).Times(3)
			err := cloudA.cloud.CreateVpcPeering(cloudB.cloud, api.NewLoggingReporter())
			Expect(err).To(HaveOccurred())
		})
		It("creates the VPC peering successfully", func() {
			mockDescribeVPCs(cloudA, cloudB)
			mockDescribeVPCs(cloudA, cloudB)
			mockRequestPeering(cloudA)
			mockAcceptPeering(cloudB)
			mockCreateRoutes(cloudA, cloudB)
			err := cloudA.cloud.CreateVpcPeering(cloudB.cloud, api.NewLoggingReporter())
			Expect(err).ToNot(HaveOccurred())
		})
	})
}

func testRequestPeering() {
	cloudA := newCloudTestDriver(infraID, region)
	cloudB := newCloudTestDriver(targetInfraID, targetRegion)
	vpcA := "vpc-a"
	vpcB := "vpc-b"
	vpcPeeringID := "peering-id"
	var awsCloudA, awsCloudB *awsCloud
	var ok bool

	_ = Describe("Validate Peering request", func() {
		BeforeEach(func() {
			awsCloudA, ok = cloudA.cloud.(*awsCloud)
			Expect(ok).To(BeTrue())
			awsCloudB, ok = cloudB.cloud.(*awsCloud)
			Expect(ok).To(BeTrue())
		})
		When("Request a Peering Request", func() {
			It("can request it", func() {
				cloudA.awsClient.EXPECT().CreateVpcPeeringConnection(context.TODO(), gomock.Any()).
					Return(&ec2.CreateVpcPeeringConnectionOutput{
						VpcPeeringConnection: &types.VpcPeeringConnection{
							RequesterVpcInfo: &types.VpcPeeringConnectionVpcInfo{
								VpcId:  &vpcA,
								Region: &awsCloudA.region,
							},
							VpcPeeringConnectionId: &vpcPeeringID,
						},
					}, nil)

				vpcPeering, err := awsCloudA.requestPeering(vpcA, vpcB, awsCloudB, api.NewLoggingReporter())

				Expect(err).To(BeNil())
				Expect(vpcPeering).NotTo(BeNil())
				Expect(*vpcPeering.RequesterVpcInfo.Region).To(Equal(awsCloudA.region))
				Expect(*vpcPeering.RequesterVpcInfo.VpcId).To(Equal(vpcA))
			})
			It("can't request a VPC peering", func() {
				errMsg := "unable to request VPC peering"

				cloudA.awsClient.EXPECT().CreateVpcPeeringConnection(context.TODO(), gomock.Any()).
					Return(nil, errors.New(errMsg))

				vpcPeering, err := awsCloudA.requestPeering(vpcA, vpcB, awsCloudB, api.NewLoggingReporter())

				Expect(err).Should(MatchError(MatchRegexp(errMsg)))
				Expect(vpcPeering).To(BeNil())
			})
		})
	})
}

func testAcceptPeering() {
	cloudA := newCloudTestDriver(infraID, region)
	peeringID := "peer-id"
	var awsCloudA *awsCloud
	var ok bool

	_ = Describe("Validate Accept peering process", func() {
		BeforeEach(func() {
			awsCloudA, ok = cloudA.cloud.(*awsCloud)
			Expect(ok).To(BeTrue())
		})
		When("Trying to accept a Peering Request", func() {
			It("is accepted", func() {
				cloudA.awsClient.EXPECT().AcceptVpcPeeringConnection(context.TODO(), gomock.Any()).
					Return(nil, nil)

				err := awsCloudA.acceptPeering(&peeringID, api.NewLoggingReporter())
				Expect(err).To(BeNil())
			})
			It("is not accepted", func() {
				cloudA.awsClient.EXPECT().AcceptVpcPeeringConnection(context.TODO(), gomock.Any()).
					Return(nil, errors.New("Accept Peering Error"))

				err := awsCloudA.acceptPeering(&peeringID, api.NewLoggingReporter())
				Expect(err).NotTo(BeNil())
			})
		})
	})
}

func testCreateRoutesForPeering() {
	cloudA := newCloudTestDriver(infraID, region)
	cloudB := newCloudTestDriver(targetInfraID, targetRegion)
	vpcA := "vpc-a"
	vpcB := "vpc-b"
	peeringID := "peer-id"
	peering := types.VpcPeeringConnection{
		AccepterVpcInfo: &types.VpcPeeringConnectionVpcInfo{
			VpcId: &vpcA,
		},
		RequesterVpcInfo: &types.VpcPeeringConnectionVpcInfo{
			VpcId: &vpcB,
		},
		VpcPeeringConnectionId: &peeringID,
	}
	var awsCloudA, awsCloudB *awsCloud
	var ok bool

	_ = Describe("Validate error input", func() {
		BeforeEach(func() {
			awsCloudA, ok = cloudA.cloud.(*awsCloud)
			Expect(ok).To(BeTrue())
			awsCloudB, ok = cloudB.cloud.(*awsCloud)
			Expect(ok).To(BeTrue())
		})
		When("Create Routes For peering", func() {
			It("can create them", func() {
				cloudA.awsClient.EXPECT().DescribeRouteTables(context.TODO(), gomock.Any()).
					Return(getRouteTableFor(vpcA), nil)
				cloudB.awsClient.EXPECT().DescribeRouteTables(context.TODO(), gomock.Any()).
					Return(getRouteTableFor(vpcB), nil)
				cloudA.awsClient.EXPECT().CreateRoute(context.TODO(), gomock.Any()).
					Return(nil, nil)
				cloudB.awsClient.EXPECT().CreateRoute(context.TODO(), gomock.Any()).
					Return(nil, nil)

				err := awsCloudA.createRoutesForPeering(awsCloudB, vpcA, vpcB, &peering, api.NewLoggingReporter())

				Expect(err).To(BeNil())
			})
			It("Can't create route on requester", func() {
				cloudA.awsClient.EXPECT().DescribeRouteTables(context.TODO(), gomock.Any()).
					Return(getRouteTableFor(vpcA), nil)
				cloudA.awsClient.EXPECT().CreateRoute(context.TODO(), gomock.Any()).
					Return(nil, errors.New("Can't create route"))

				err := awsCloudA.createRoutesForPeering(awsCloudB, vpcA, vpcB, &peering, api.NewLoggingReporter())

				Expect(err).NotTo(BeNil())
				Expect(err).Should(MatchError(MatchRegexp("unable to create route for " + vpcA)))
			})
			It("Can't create route on accepter", func() {
				cloudA.awsClient.EXPECT().DescribeRouteTables(context.TODO(), gomock.Any()).
					Return(getRouteTableFor(vpcA), nil)
				cloudB.awsClient.EXPECT().DescribeRouteTables(context.TODO(), gomock.Any()).
					Return(getRouteTableFor(vpcB), nil)
				cloudA.awsClient.EXPECT().CreateRoute(context.TODO(), gomock.Any()).
					Return(nil, nil)
				cloudB.awsClient.EXPECT().CreateRoute(context.TODO(), gomock.Any()).
					Return(nil, errors.New("Can't create route"))

				err := awsCloudA.createRoutesForPeering(awsCloudB, vpcA, vpcB, &peering, api.NewLoggingReporter())

				Expect(err).NotTo(BeNil())
				Expect(err).Should(MatchError(MatchRegexp("unable to create route for " + vpcB)))
			})
			It("Can't get Requester Route Table", func() {
				cloudA.awsClient.EXPECT().DescribeRouteTables(context.TODO(), gomock.Any()).
					Return(nil, errors.New("unable to create route for "+vpcA))

				err := awsCloudA.createRoutesForPeering(awsCloudB, vpcA, vpcB, &peering, api.NewLoggingReporter())

				Expect(err).NotTo(BeNil())
				Expect(err).Should(MatchError(MatchRegexp("unable to create route for " + vpcA)))
			})
			It("Can't get Accepter Route Table", func() {
				cloudA.awsClient.EXPECT().DescribeRouteTables(context.TODO(), gomock.Any()).
					Return(getRouteTableFor(vpcA), nil)
				cloudA.awsClient.EXPECT().CreateRoute(context.TODO(), gomock.Any()).
					Return(nil, nil)
				cloudB.awsClient.EXPECT().DescribeRouteTables(context.TODO(), gomock.Any()).
					Return(nil, errors.New("unable to create route for "+vpcB))

				err := awsCloudA.createRoutesForPeering(awsCloudB, vpcA, vpcB, &peering, api.NewLoggingReporter())

				Expect(err).NotTo(BeNil())
				Expect(err).Should(MatchError(MatchRegexp("unable to create route for " + vpcB)))
			})
		})
	})
}

func testGetRouteTableID() {
	cloudA := newCloudTestDriver(infraID, region)
	vpcA := "vpc-a"
	var awsCloudA *awsCloud
	var ok bool

	_ = Describe("Test Get Route Table ID", func() {
		BeforeEach(func() {
			awsCloudA, ok = cloudA.cloud.(*awsCloud)
			Expect(ok).To(BeTrue())
		})
		When("Trying to get Route Table ID", func() {
			It("returns correct Route Table ID", func() {
				cloudA.awsClient.EXPECT().DescribeRouteTables(context.TODO(), gomock.Any()).
					Return(getRouteTableFor(vpcA), nil)

				rtID, err := awsCloudA.getRouteTableID(vpcA, api.NewLoggingReporter())

				Expect(err).To(BeNil())
				Expect(rtID).ToNot(BeNil())
				Expect(rtID).ToNot(Equal(vpcA + "-rt"))
			})
		})
		It("can't return Route Table ID", func() {
			errMsg := "Route Table not Found"

			cloudA.awsClient.EXPECT().DescribeRouteTables(context.TODO(), gomock.Any()).
				Return(nil, errors.New(errMsg))

			rtID, err := awsCloudA.getRouteTableID("", api.NewLoggingReporter())

			Expect(err).ToNot(BeNil())
			Expect(err).Should(MatchError(MatchRegexp(errMsg)))
			Expect(rtID).To(BeNil())
		})
	})
}

func testCleanupVpcPeering() {
	cloudA := newCloudTestDriver(infraID, region)
	cloudB := newCloudTestDriver(targetInfraID, targetRegion)
	var awsCloudA *awsCloud
	var awsCloudB *awsCloud
	var ok bool

	_ = Describe("Test Clean up VPC Peering", func() {
		BeforeEach(func() {
			awsCloudA, ok = cloudA.cloud.(*awsCloud)
			Expect(ok).To(BeTrue())
			awsCloudB, ok = cloudB.cloud.(*awsCloud)
			Expect(ok).To(BeTrue())
		})
		When("Trying to get vpcIDs", func() {
			It("the source VpcID returns an error", func() {
				cloudA.awsClient.EXPECT().DescribeVpcs(context.TODO(), gomock.Any()).
					Return(nil, errors.New("unable to describe source vpcs"))

				err := awsCloudA.cleanupVpcPeering(awsCloudB, api.NewLoggingReporter())

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(MatchRegexp("unable to retrieve source VPC ID")))
			})
			It("the target VpcID returns an error", func() {
				cloudA.awsClient.EXPECT().
					DescribeVpcs(context.TODO(), gomock.Any()).
					Return(getVpcOutputFor("vpc-a", "10.0.0.0/16"), nil)
				cloudB.awsClient.EXPECT().DescribeVpcs(context.TODO(), gomock.Any()).
					Return(nil, errors.New("unable to describe target vpcs"))

				err := awsCloudA.cleanupVpcPeering(awsCloudB, api.NewLoggingReporter())

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(MatchRegexp("unable to retrieve target VPC ID")))
			})
		})
		When("Trying to describe VPC peering connections", func() {
			It("returns an error", func() {
				mockDescribeVPCs(cloudA, cloudB)
				cloudA.awsClient.EXPECT().DescribeVpcPeeringConnections(context.TODO(), gomock.Any()).
					Return(nil, errors.New("unable to describe source vpcs"))

				err := awsCloudA.cleanupVpcPeering(awsCloudB, api.NewLoggingReporter())

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(MatchRegexp("unable to retrieve VPC Peering Connections")))
			})
			It("does not return any VPC Peering connection", func() {
				mockDescribeVPCs(cloudA, cloudB)
				cloudA.awsClient.EXPECT().DescribeVpcPeeringConnections(context.TODO(), gomock.Any()).
					Return(&ec2.DescribeVpcPeeringConnectionsOutput{
						VpcPeeringConnections: []types.VpcPeeringConnection{},
					}, nil)

				err := awsCloudA.cleanupVpcPeering(awsCloudB, api.NewLoggingReporter())

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(MatchRegexp("expecting exactly 1 VpcPeeringConnections, got 0")))
			})
		})
		When("Trying to delete routes", func() {
			It("returns an error", func() {
				vpcPeeringID := "vpc-peering-id"
				mockDescribeVPCs(cloudA, cloudB)
				cloudA.awsClient.EXPECT().DescribeVpcPeeringConnections(context.TODO(), gomock.Any()).
					Return(&ec2.DescribeVpcPeeringConnectionsOutput{
						VpcPeeringConnections: []types.VpcPeeringConnection{
							{
								VpcPeeringConnectionId: &vpcPeeringID,
							},
						},
					}, nil)
				cloudA.awsClient.EXPECT().DescribeRouteTables(context.TODO(), gomock.Any()).
					Return(nil, errors.New("unable to describe route tables"))

				err := awsCloudA.cleanupVpcPeering(awsCloudB, api.NewLoggingReporter())

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(MatchRegexp("unable to get route tables for vpc-a")))
			})
			It("does not return any VPC Peering connection", func() {
				mockDescribeVPCs(cloudA, cloudB)
				cloudA.awsClient.EXPECT().DescribeVpcPeeringConnections(context.TODO(), gomock.Any()).
					Return(&ec2.DescribeVpcPeeringConnectionsOutput{
						VpcPeeringConnections: []types.VpcPeeringConnection{},
					}, nil)

				err := awsCloudA.cleanupVpcPeering(awsCloudB, api.NewLoggingReporter())

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(MatchRegexp("expecting exactly 1 VpcPeeringConnections, got 0")))
			})
		})
		When("Trying to delete vpc peering connection", func() {
			It("returns an error", func() {
				vpcPeeringID := "vpc-peering-id"
				cidrBlock := "10.1.0.0/16"
				srcVpcID := "src-vpc-id"
				targetVpcID := "target-vpc-id"
				mockDescribeVPCs(cloudA, cloudB)
				cloudA.awsClient.EXPECT().DescribeVpcPeeringConnections(context.TODO(), gomock.Any()).
					Return(&ec2.DescribeVpcPeeringConnectionsOutput{
						VpcPeeringConnections: []types.VpcPeeringConnection{
							{
								VpcPeeringConnectionId: &vpcPeeringID,
								AccepterVpcInfo: &types.VpcPeeringConnectionVpcInfo{
									VpcId:     &targetVpcID,
									CidrBlock: &cidrBlock,
								},
								RequesterVpcInfo: &types.VpcPeeringConnectionVpcInfo{
									VpcId:     &srcVpcID,
									CidrBlock: &cidrBlock,
								},
							},
						},
					}, nil)
				mockGetRouteTableID(cloudA)
				mockGetRouteTableID(cloudB)
				cloudA.awsClient.EXPECT().DeleteRoute(context.TODO(), gomock.Any()).
					Return(nil, nil)
				cloudB.awsClient.EXPECT().DeleteRoute(context.TODO(), gomock.Any()).
					Return(nil, nil)
				cloudA.awsClient.EXPECT().DeleteVpcPeeringConnection(context.TODO(), gomock.Any()).
					Return(nil, errors.New("unable to remove vpc peering connection"))

				err := awsCloudA.cleanupVpcPeering(awsCloudB, api.NewLoggingReporter())

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(MatchRegexp("unable to delete VPC Peering Connection vpc-peering-id")))
			})
			It("succeeds", func() {
				vpcPeeringID := "vpc-peering-id"
				cidrBlock := "10.1.0.0/16"
				srcVpcID := "src-vpc-id"
				targetVpcID := "target-vpc-id"
				mockDescribeVPCs(cloudA, cloudB)
				cloudA.awsClient.EXPECT().DescribeVpcPeeringConnections(context.TODO(), gomock.Any()).
					Return(&ec2.DescribeVpcPeeringConnectionsOutput{
						VpcPeeringConnections: []types.VpcPeeringConnection{
							{
								VpcPeeringConnectionId: &vpcPeeringID,
								AccepterVpcInfo: &types.VpcPeeringConnectionVpcInfo{
									VpcId:     &targetVpcID,
									CidrBlock: &cidrBlock,
								},
								RequesterVpcInfo: &types.VpcPeeringConnectionVpcInfo{
									VpcId:     &srcVpcID,
									CidrBlock: &cidrBlock,
								},
							},
						},
					}, nil)
				mockGetRouteTableID(cloudA)
				mockGetRouteTableID(cloudB)
				cloudA.awsClient.EXPECT().DeleteRoute(context.TODO(), gomock.Any()).
					Return(nil, nil)
				cloudB.awsClient.EXPECT().DeleteRoute(context.TODO(), gomock.Any()).
					Return(nil, nil)
				cloudA.awsClient.EXPECT().DeleteVpcPeeringConnection(context.TODO(), gomock.Any()).
					Return(nil, nil)

				err := awsCloudA.cleanupVpcPeering(awsCloudB, api.NewLoggingReporter())

				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
}

func testDeleteVpcPeeringRoutes() {
	cloudA := newCloudTestDriver(infraID, region)
	cloudB := newCloudTestDriver(targetInfraID, targetRegion)
	var awsCloudA *awsCloud
	var awsCloudB *awsCloud
	var ok bool

	_ = Describe("Test Delete VPC Peering routes", func() {
		BeforeEach(func() {
			awsCloudA, ok = cloudA.cloud.(*awsCloud)
			Expect(ok).To(BeTrue())
			awsCloudB, ok = cloudB.cloud.(*awsCloud)
			Expect(ok).To(BeTrue())
		})
		When("Trying to get source Route Table ID", func() {
			It("returns an error", func() {
				srcVpcID := "src-vpc-id"
				targetVpcID := "target-vpc-id"

				cloudA.awsClient.EXPECT().DescribeRouteTables(context.TODO(), gomock.Any()).
					Return(nil, errors.New("unable to describe route table"))

				err := awsCloudA.deleteVpcPeeringRoutes(awsCloudB, srcVpcID, targetVpcID,
					&types.VpcPeeringConnection{}, api.NewLoggingReporter())

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(MatchRegexp("unable to get route tables for src-vpc-id")))
			})
		})
		When("Trying to delete the source Route", func() {
			It("returns an error", func() {
				srcVpcID := "src-vpc-id"
				targetVpcID := "target-vpc-id"
				cidrBlock := "10.1.0.0/16"

				mockGetRouteTableID(cloudA)
				cloudA.awsClient.EXPECT().DeleteRoute(context.TODO(), gomock.Any()).
					Return(nil, errors.New("unable to delete route"))

				err := awsCloudA.deleteVpcPeeringRoutes(awsCloudB, srcVpcID, targetVpcID,
					&types.VpcPeeringConnection{
						RequesterVpcInfo: &types.VpcPeeringConnectionVpcInfo{
							CidrBlock: &cidrBlock,
						},
						AccepterVpcInfo: &types.VpcPeeringConnectionVpcInfo{
							CidrBlock: &cidrBlock,
						},
					},
					api.NewLoggingReporter())

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(MatchRegexp("unable to delete route for src-vpc-id")))
			})
		})
		When("Trying to get target Route Table ID", func() {
			It("returns an error", func() {
				srcVpcID := "src-vpc-id"
				targetVpcID := "target-vpc-id"
				cidrBlock := "10.1.0.0/16"

				mockGetRouteTableID(cloudA)
				cloudA.awsClient.EXPECT().DeleteRoute(context.TODO(), gomock.Any()).
					Return(nil, nil)

				cloudB.awsClient.EXPECT().DescribeRouteTables(context.TODO(), gomock.Any()).
					Return(nil, errors.New("unable to describe route table"))

				err := awsCloudA.deleteVpcPeeringRoutes(awsCloudB, srcVpcID, targetVpcID,
					&types.VpcPeeringConnection{
						RequesterVpcInfo: &types.VpcPeeringConnectionVpcInfo{
							CidrBlock: &cidrBlock,
						},
						AccepterVpcInfo: &types.VpcPeeringConnectionVpcInfo{
							CidrBlock: &cidrBlock,
						},
					},
					api.NewLoggingReporter())

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(MatchRegexp("unable to get route tables for target-vpc-id")))
			})
		})
		When("Trying to delete the target Route", func() {
			It("returns an error", func() {
				srcVpcID := "src-vpc-id"
				targetVpcID := "target-vpc-id"
				cidrBlock := "10.1.0.0/16"

				mockGetRouteTableID(cloudA)
				cloudA.awsClient.EXPECT().DeleteRoute(context.TODO(), gomock.Any()).
					Return(nil, nil)
				mockGetRouteTableID(cloudB)
				cloudB.awsClient.EXPECT().DeleteRoute(context.TODO(), gomock.Any()).
					Return(nil, errors.New("unable to delete target route"))
				err := awsCloudA.deleteVpcPeeringRoutes(awsCloudB, srcVpcID, targetVpcID,
					&types.VpcPeeringConnection{
						RequesterVpcInfo: &types.VpcPeeringConnectionVpcInfo{
							CidrBlock: &cidrBlock,
						},
						AccepterVpcInfo: &types.VpcPeeringConnectionVpcInfo{
							CidrBlock: &cidrBlock,
						},
					},
					api.NewLoggingReporter())

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(MatchRegexp("unable to delete route for target-vpc-id")))
			})
		})
		When("Trying to delete all vpc peering routes", func() {
			It("succeeds", func() {
				srcVpcID := "src-vpc-id"
				targetVpcID := "target-vpc-id"
				cidrBlock := "10.1.0.0/16"
				vpcID := "vpc-id"

				mockGetRouteTableID(cloudA)
				cloudA.awsClient.EXPECT().DeleteRoute(context.TODO(), gomock.Any()).
					Return(nil, nil)
				mockGetRouteTableID(cloudB)
				cloudB.awsClient.EXPECT().DeleteRoute(context.TODO(), gomock.Any()).
					Return(nil, nil)
				err := awsCloudA.deleteVpcPeeringRoutes(awsCloudB, srcVpcID, targetVpcID,
					&types.VpcPeeringConnection{
						VpcPeeringConnectionId: &vpcID,
						RequesterVpcInfo: &types.VpcPeeringConnectionVpcInfo{
							CidrBlock: &cidrBlock,
						},
						AccepterVpcInfo: &types.VpcPeeringConnectionVpcInfo{
							CidrBlock: &cidrBlock,
						},
					},
					api.NewLoggingReporter())

				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
}

func HasCreateVpcPeeringConnectionInput(vpcID, peerVpcID, peerRegion, nameTagValue string) gomock.Matcher {
	return createVpcPeeringConnectionInputMatcher{
		vpcID,
		peerVpcID,
		peerRegion,
		nameTagValue,
	}
}

type createVpcPeeringConnectionInputMatcher struct {
	VpcID        string `json:"vpc_id"`
	PeerVpcID    string `json:"peer_vpc_id"`
	PeerRegion   string `json:"peer_region"`
	NameTagValue string `json:"name_tag_value"`
}

func (e createVpcPeeringConnectionInputMatcher) Matches(arg interface{}) bool {
	input, _ := arg.(*ec2.CreateVpcPeeringConnectionInput)

	return *input.VpcId == e.VpcID &&
		*input.PeerVpcId == e.PeerVpcID &&
		*input.PeerRegion == e.PeerRegion &&
		len(input.TagSpecifications) == 1 &&
		input.TagSpecifications[0].ResourceType == types.ResourceTypeVpcPeeringConnection &&
		len(input.TagSpecifications[0].Tags) == 1 &&
		*input.TagSpecifications[0].Tags[0].Value == *ec2Tag("Name", e.NameTagValue).Value
}

func (e createVpcPeeringConnectionInputMatcher) String() string {
	return fmt.Sprint(json.Marshal(e))
}

func getVpcOutputFor(id, cidrBlock string) *ec2.DescribeVpcsOutput {
	return &ec2.DescribeVpcsOutput{
		Vpcs: []types.Vpc{
			{
				VpcId:     &id,
				CidrBlock: &cidrBlock,
			},
		},
	}
}

func mockDescribeVPCs(cloudA, cloudB *cloudTestDriver) {
	cloudA.awsClient.EXPECT().
		DescribeVpcs(context.TODO(), gomock.Any()).
		Return(getVpcOutputFor("vpc-a", "10.0.0.0/16"), nil).
		Times(1)

	cloudB.awsClient.EXPECT().
		DescribeVpcs(context.TODO(), gomock.Any()).
		Return(getVpcOutputFor("vpc-b", "10.1.0.0/16"), nil).
		Times(1)
}

func mockRequestPeering(cloud *cloudTestDriver) {
	peeringID := "test-peering-id"
	cidrBlock := "10.1.0.0/16"

	cloud.awsClient.EXPECT().CreateVpcPeeringConnection(context.TODO(), gomock.Any()).
		Return(&ec2.CreateVpcPeeringConnectionOutput{
			VpcPeeringConnection: &types.VpcPeeringConnection{
				VpcPeeringConnectionId: &peeringID,
				AccepterVpcInfo: &types.VpcPeeringConnectionVpcInfo{
					CidrBlock: &cidrBlock,
				},
				RequesterVpcInfo: &types.VpcPeeringConnectionVpcInfo{
					CidrBlock: &cidrBlock,
				},
			},
		}, nil)
}

func mockAcceptPeering(cloud *cloudTestDriver) {
	cloud.awsClient.EXPECT().AcceptVpcPeeringConnection(context.TODO(), gomock.Any()).
		Return(nil, nil)
}

func mockCreateRoutes(cloudA, cloudB *cloudTestDriver) {
	cloudARouteTableID := "cloud-a-routetable"
	cloudBRouteTableID := "cloud-b-routetable"

	cloudA.awsClient.EXPECT().DescribeRouteTables(context.TODO(), gomock.Any()).
		Return(&ec2.DescribeRouteTablesOutput{
			RouteTables: []types.RouteTable{
				{
					RouteTableId: &cloudARouteTableID,
				},
			},
		}, nil)
	cloudB.awsClient.EXPECT().DescribeRouteTables(context.TODO(), gomock.Any()).
		Return(&ec2.DescribeRouteTablesOutput{
			RouteTables: []types.RouteTable{
				{
					RouteTableId: &cloudBRouteTableID,
				},
			},
		}, nil)

	cloudA.awsClient.EXPECT().CreateRoute(context.TODO(), gomock.Any()).
		Return(nil, nil)
	cloudB.awsClient.EXPECT().CreateRoute(context.TODO(), gomock.Any()).
		Return(nil, nil)
}

func mockGetRouteTableID(cloud *cloudTestDriver) {
	routeTableID := "route-table-id"

	cloud.awsClient.EXPECT().DescribeRouteTables(context.TODO(), gomock.Any()).
		Return(&ec2.DescribeRouteTablesOutput{
			RouteTables: []types.RouteTable{
				{
					RouteTableId: &routeTableID,
				},
			},
		}, nil)
}
