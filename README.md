# ECR Continuous Image Scanning

This repo shows how to use the [ECR image scanning](https://docs.aws.amazon.com/AmazonECR/latest/userguide/security.html) feature
for a continuous scan, that is, scanning images on a regular basis. We will walk you through the setup and usage of this demo.

## Installation

Prerequisites: SAM, `aws` CLI 

Preparing the S3 buckets:

```sh
export ECR_SCAN_SVC_BUCKET=ecr-continuous-scan-svc
export ECR_SCAN_CONFIG_BUCKET=ecr-continuous-scan-config

aws s3api create-bucket \
            --bucket $ECR_SCAN_SVC_BUCKET \
            --create-bucket-configuration LocationConstraint=$(aws configure get region) \
            --region $(aws configure get region)

aws s3api create-bucket \
            --bucket $ECR_SCAN_CONFIG_BUCKET \
            --create-bucket-configuration LocationConstraint=$(aws configure get region) \
            --region $(aws configure get region)
```

Then `make deploy` ...

## Architecture

Overall: Lambda functions, HTTP API

### Scan configurations

```json
{
    "region": "us-west-2",
    "registry": "148658015984",
    "repository": "amazonlinux",
    "tags": [
        "2018.03"
    ]
}
```

Note that `tags` is optional and if not provided, all tags of the `repository` will be scanned. 

### API

The following HTTP API is exposed:

Scan configurations:

* `GET configs/` … lists all registered scan configurations, returns JSON
* `POST configs/` … adds a scan configuration, returns scan ID
* `DELETE configs/{scanid}` … removes a registered scan configuration by scan ID or `404` if it doesn't exist

Scan findings:

* `GET summary/` … provides high-level summary of findings across all registered scan configurations
* `GET findings/{scanid}` … provides detailed findings on a scan configuration bases, returns an Atom feed


## Usage

```sh
export ECRSCANAPI_URL=$(aws cloudformation describe-stacks --stack-name ecr-continuous-scan | jq '.Stacks[].Outputs[] | select(.OutputKey=="ECRScanAPIEndpoint").OutputValue' -r)
```

Managing scan configurations:

```sh
curl $ECRSCANAPI_URL/configs/

curl -s --header "Content-Type: application/json" --request POST --data @scan-config-amazonlinux.json $ECRSCANAPI_URL/configs/

curl --request DELETE $ECRSCANAPI_URL/configs/e7c3b83c-b995-44d3-942b-8b1001f33ae
```

Manage scan results:

```sh
curl $ECRSCANAPI_URL/summary

curl $ECRSCANAPI_URL/findings/e7c3b83c-b995-44d3-942b-8b1001f33ae
```