# ECR Continuous Image Scanning

This repo shows how to use the [ECR image scanning](https://docs.aws.amazon.com/AmazonECR/latest/userguide/security.html) feature
for a continuous scan, that is, scanning images on a regular basis. We will walk you through the setup and usage of this demo.

## Installation

In order to build and deploy the service, clone this repo and make sure you've got the following available, locally:

- The [aws](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html) CLI
- The [SAM CLI](https://github.com/awslabs/aws-sam-cli)
- Go 1.12 or above

Additionally, having [jq](https://stedolan.github.io/jq/download/) installed it recommended.

Preparing the S3 buckets (make sure that you pick different names for the `ECR_SCAN_*` buckets):

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

Make sure that you have the newest [Go SDK](https://aws.amazon.com/sdk-for-go/) installed, 
supporting the image scanning feature. In addition, you need to `go get https://github.com/gorilla/feeds`
as the one other dependency outside of the standard library. Then execute:

```sh
make deploy
``` 

which will build the binaries and deploy the Lambda functions. 

You're now ready to use the demo.


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