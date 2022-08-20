module github.com/DentonGentry/tailscale-aws-host-acl-updater

go 1.19

require (
	github.com/aws/aws-lambda-go v1.34.1
	github.com/aws/aws-sdk-go-v2 v1.16.11
	github.com/aws/aws-sdk-go-v2/config v1.16.1
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.52.1
	github.com/aws/aws-sdk-go-v2/service/rds v1.23.6
	github.com/tailscale/hujson v0.0.0-20220630195928-54599719472f
)

require (
	github.com/aws/aws-sdk-go-v2/credentials v1.12.13 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.12.12 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.18 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.4.12 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.19 // indirect
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2 v1.18.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.9.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.11.16 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.16.13 // indirect
	github.com/aws/smithy-go v1.12.1 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/stretchr/testify v1.7.4 // indirect
)
