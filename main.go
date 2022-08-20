package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	elb "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/rds"
)

var (
	tsApiKey        string
	tailnet         string
	tsControlServer string
)

func HandleRequest(ctx context.Context, event *events.CloudWatchEvent) {
	hosts := tagsHandler(ctx, event)
	if len(hosts) > 0 {
		updateHosts(ctx, hosts)
	}
}

func main() {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}
	ec2Client = ec2.NewFromConfig(cfg)
	rdsClient = rds.NewFromConfig(cfg)
	elbClient = elb.NewFromConfig(cfg)

	tailnet = os.Getenv("TAILSCALE_TAILNET")
	tsControlServer = os.Getenv("TAILSCALE_CONTROL_SERVER")
	if tsControlServer == "" {
		tsControlServer = "https://login.tailscale.com"
	}

	tsApiKey, err = getApiKey()
	if err != nil {
		log.Fatalf("getApiKey failed: %v", err)
	}

	lambda.Start(HandleRequest)
}
