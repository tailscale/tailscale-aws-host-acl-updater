package main

import (
	"bytes"
	"context"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/tailscale/hujson"
)

var (
	ec2Client       *ec2.Client
	rdsClient       *rds.Client
	tsApiKey        string
	tailnet         string
	tsControlServer string
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

func handler(ctx context.Context, event events.CloudWatchEvent) {
	hosts := make(map[string]string)
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
		}
	}

	if len(hosts) > 0 {
		updateHosts(ctx, hosts)
	}
}

func getAcls() hujson.Value {
	req, err := http.NewRequest("GET", tsControlServer+"/api/v2/tailnet/"+tailnet+"/acl", nil)
	if err != nil {
		log.Printf("http.NewRequest(GET ACLs) failed: %v", err)
		return hujson.Value{}
	}
	req.SetBasicAuth(tsApiKey, "")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("http.Client(GET ACLs) failed: %v", err)
		return hujson.Value{}
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("ioutil.ReadAll(GET ACLs) failed: %v", err)
		return hujson.Value{}
	}

	value, err := hujson.Parse(body)
	if err != nil {
		log.Printf("hujson.Parse(GET ACLs) failed: %v", err)
		return hujson.Value{}
	}

	return value
}

func putAcls(acls hujson.Value) {
	url := tsControlServer + "/api/v2/tailnet/" + tailnet + "/acl"
	req, err := http.NewRequest("POST", url, bytes.NewBufferString(acls.String()))
	if err != nil {
		log.Printf("http.NewRequest(POST ACLs) failed: %v", err)
		return
	}
	req.SetBasicAuth(tsApiKey, "")
	req.Header.Set("Content-Type", "application/hujson")

	_, err = http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("http.Client(POST ACLs) failed: %v", err)
		return
	}
}

func updateHosts(ctx context.Context, update map[string]string) {
	changed := false
	acls := getAcls()

	for key, value := range update {
		patch := `[{ "op": "replace", "path": "/Hosts/` + key + `", "value": "` + value + `" }]`
		err := acls.Patch([]byte(patch))
		if err == nil {
			changed = true
		}
	}

	if changed {
		putAcls(acls)
	}
}

func main() {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}
	ec2Client = ec2.NewFromConfig(cfg)
	rdsClient = rds.NewFromConfig(cfg)

	tsApiKey = os.Getenv("TAILSCALE_API_KEY")
	tailnet = os.Getenv("TAILSCALE_TAILNET")
	tsControlServer = os.Getenv("TAILSCALE_CONTROL_SERVER")
	if tsControlServer == "" {
		tsControlServer = "https://login.tailscale.com"
	}

	lambda.Start(handler)
}
