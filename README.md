# Tailscale AWS hostname monitor

https://tailscale.com

## Overview

This repository contains a set of AWS Lambda functions intended to:
- receive notifications of changes in AWS tags across a number of
  AWS services
- check for an AWS tag named "ts-hostname"
- automatically populate the `Hosts` section of a Tailscale ACL
  Policy file to update the IP address of that hostname

This is intended to be used in conjunction with a Tailscale
[subnet router](https://tailscale.com/kb/1019/subnets/) offering
a Tailscale route to a private AWS VPC. ACLs can be defined for
the private 172.16.x.y VPC IP addresses, relying on this Lambda
function to update the IP address if the AWS instance is replaced.

## Setup

### Step 1: Lambda function
Create a Lambda function in the AWS Console in the AWS region where
the EC2/RDS/etc resources used with Tailscale are located.

CloudWatch events can only be sent to a Lambda function in the same
region. A Lambda will be needed in each region where there are
resources to be tracked.


### Step 2: Permissions
In Lambda > Configuration > Permissions, the ARN of the role created for the
Lambda function will be shown. It needs to be given several permissions.

a. It needs to be able to read EC2 Instance Details. Attach the
`AmazonEC2ReadOnlyAccess` policy to the role.
 
b. It needs to be able to get RDS instance details and tags. Create
an inline policy of:
```
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "AllowRDSTagsRead",
            "Effect": "Allow",
            "Action": [
                "rds:DescribeDBInstances",
		"rds:ListTagsForResource"
            ],
            "Resource": "*"
        }
    ]
}
```

### Step 3: Configuration
In Lambda > Configuration > Environment variables, create the following variable:
- `TAILSCALE_TAILNET`: the name of the tailnet, such as `example.com` or `octocat.github`


In the AWS Secrets Manager create a secret containing a Key/Value pair:
- Key: `API_KEY`
- Value: the `tskey-...` of a [Tailscale API key](https://tailscale.com/kb/1101/api/).

In the Replicate Secret section you can replicate the same secret across all regions
where you intend to run this Lambda function. All of them can use the same API Key.

We suggest naming the secret `ts-acl-hostname-updater`, but you may choose
whatever naming convention you prefer by setting the `SECRET_NAME` environment variable.

In the role created for the Lambda function (in Lambda > Configuration > Permissions) add
another inline policy to access the secret:
```
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "secretsmanager:GetSecretValue"
            ],
            "Resource": [
                "ARN_OF_SECRET"
            ]
        }
    ]
}
```

### Step 4: EventBridge
Create an EventBridge event with pattern:
```
{
  "source": ["aws.tag"],
  "detail-type": ["Tag Change on Resource"],
  "detail": {
    "service": ["ec2", "rds", "elasticloadbalancing"]
  }
}
```
This will run the Lambda function whenever tags are changed on an EC2 or RDS instance.

Create a second EventBridge rule with the pattern:
```
{
  "source": ["aws.rds"],
  "detail-type": ["RDS DB Instance Event"]
}
```
This will run the Lambda function whenever the RDS instance reboots or otherwise
changes its running state.

Create a third EventBridge rule with the pattern:
```
{
  "source": ["aws.elasticloadbalancing"],
  "detail-type": ["AWS API Call via CloudTrail"],
  "detail": {
    "eventSource": ["elasticloadbalancing.amazonaws.com"],
    "eventName": ["CreateLoadBalancer"]
  }
}
```
This will run the Lambda function whenever a new Load Balancer instance is created.
Unfortunately AWS does not generate CloudWatch events for other Load Balancer state
changes, we cannot automatically run the Lambda.


### Step 5: Add `ts-hostname` AWS Tags
For any AWS resource for which you wish this Lambda function to keep the IP
address up to date, two things need to be done:
1. Add an AWS Tag with key `ts-hostname` and value of the hostname which it is to maintain.
   This is an _AWS_ Tag, not a Tag in the Tailscale ACL file.
1. Add an initial entry for the hostname in the [Tailscale ACLs Hosts section](https://login.tailscale.com/admin/acls).
   This Lambda function will update a `Hosts` entry which is already present. It will not
   add a new Hosts entry. This initial entry can be any valid IP address like `1.2.3.4`

## Bugs

Please file any issues about this function on
[the issue tracker](https://github.com/tailscale/tailscale/issues).

## Contributing

PRs welcome! But please file bugs. Commit messages should [reference
bugs](https://docs.github.com/en/github/writing-on-github/autolinked-references-and-urls).

We require [Developer Certificate of
Origin](https://en.wikipedia.org/wiki/Developer_Certificate_of_Origin)
`Signed-off-by` lines in commits.

## About Us

[Tailscale](https://tailscale.com/) is primarily developed by the
people at https://github.com/orgs/tailscale/people. For other contributors,
see:

* https://github.com/tailscale/tailscale/graphs/contributors
* https://github.com/tailscale/tailscale-android/graphs/contributors

## Legal

WireGuard is a registered trademark of Jason A. Donenfeld.
