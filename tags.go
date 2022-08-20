package main

import (
	"context"
	"log"
	"net"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	elb "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/rds"
)

var (
	ec2Client *ec2.Client
	elbClient *elb.Client
	rdsClient *rds.Client
)

func getEc2VpcAddress(ctx context.Context, resource string) (hostname, ip string) {
	id := strings.TrimPrefix(resource, "instance/")
	params := &ec2.DescribeInstancesInput{
		InstanceIds: []string{id},
	}
	output, err := ec2Client.DescribeInstances(ctx, params)
	if err != nil {
		log.Printf("ec2.DescribeInstances(%s) failed: %v", id, err)
		return "", ""
	}
	if len(output.Reservations) != 1 || len(output.Reservations[0].Instances) != 1 {
		return "", ""
	}

	instance := output.Reservations[0].Instances[0]
	for _, tag := range instance.Tags {
		if *tag.Key == "ts-hostname" {
			hostname = *tag.Value
			break
		}
	}
	if hostname != "" {
		ip = *instance.PrivateIpAddress
	}

	return
}

func getRdsVpcAddress(ctx context.Context, arn string) (hostname, ip string) {
	tagsParams := &rds.ListTagsForResourceInput{
		ResourceName: &arn,
	}
	tagsOutput, err := rdsClient.ListTagsForResource(ctx, tagsParams)
	if err != nil {
		log.Printf("rds.ListTagsForResource(%s) failed: %v", arn, err)
		return "", ""
	}
	instanceParams := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: &arn,
	}
	instanceOutput, err := rdsClient.DescribeDBInstances(ctx, instanceParams)
	if err != nil {
		log.Printf("rds.DescribeDBInstances(%s) failed: %v", arn, err)
		return "", ""
	}
	if len(instanceOutput.DBInstances) != 1 {
		return "", ""
	}

	rdsHostname := ""
	for _, tag := range tagsOutput.TagList {
		if *tag.Key == "ts-hostname" {
			rdsHostname = *tag.Value
			break
		}
	}
	if rdsHostname == "" {
		return "", ""
	}

	dnsName := instanceOutput.DBInstances[0].Endpoint.Address
	ips, err := net.LookupIP(*dnsName)
	if err != nil {
		log.Printf("net.LookupIP(%s) failed: %v", dnsName, err)
		return "", ""
	}
	if len(ips) < 1 {
		log.Printf("net.LookupIP(%s): no results", dnsName)
		return "", ""
	}

	ip = ips[0].String()
	hostname = rdsHostname
	return
}

func getElbVpcAddress(ctx context.Context, arn string) (hostname, ip string) {
	tagsParams := &elb.DescribeTagsInput{
		ResourceArns: []string{arn},
	}
	tagsOutput, err := elbClient.DescribeTags(ctx, tagsParams)
	if err != nil {
		log.Printf("elb.DescribeTags(%s) failed: %v", arn, err)
		return "", ""
	}

	elbParams := &elb.DescribeLoadBalancersInput{
		LoadBalancerArns: []string{arn},
	}
	instanceOutput, err := elbClient.DescribeLoadBalancers(ctx, elbParams)
	if err != nil {
		log.Printf("elb.DescribeLoadBalancers(%s) failed: %v", arn, err)
		return "", ""
	}
	if len(instanceOutput.LoadBalancers) != 1 {
		return "", ""
	}

	elbHostname := ""
	for _, desc := range tagsOutput.TagDescriptions {
		for _, tag := range desc.Tags {
			if *tag.Key == "ts-hostname" {
				elbHostname = *tag.Value
				break
			}
		}
	}
	if elbHostname == "" {
		return "", ""
	}

	dnsName := instanceOutput.LoadBalancers[0].DNSName
	ips, err := net.LookupIP(*dnsName)
	if err != nil {
		log.Printf("net.LookupIP(%s) failed: %v", *dnsName, err)
		return "", ""
	}
	if len(ips) < 1 {
		log.Printf("net.LookupIP(%s): no results", dnsName)
		return "", ""
	}

	ip = ips[0].String()
	hostname = elbHostname
	return
}

func tagsHandler(ctx context.Context, event *events.CloudWatchEvent) (hosts map[string]string) {
	hosts = make(map[string]string)
	for _, resource := range event.Resources {
		a, err := arn.Parse(resource)
		if err != nil {
			log.Printf("arn.Parse(%s) failed: %v", resource, err)
			continue
		}
		switch a.Service {
		case "ec2":
			hostname, ip := getEc2VpcAddress(ctx, a.Resource)
			if hostname != "" && ip != "" {
				hosts[hostname] = ip
			}
		case "rds":
			hostname, ip := getRdsVpcAddress(ctx, resource) // RDS APIs use full ARN
			if hostname != "" && ip != "" {
				hosts[hostname] = ip
			}
		case "elasticloadbalancing":
			hostname, ip := getElbVpcAddress(ctx, resource)
			if hostname != "" && ip != "" {
				hosts[hostname] = ip
			}
		}
	}
	return
}
