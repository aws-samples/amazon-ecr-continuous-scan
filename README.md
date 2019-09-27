# ecr-continuous-scan

Prep:

```
export ECR_SCAN_SVC_BUCKET=ecr-continuous-scan-svc
export ECR_SCAN_CONFIG_BUCKET=ecr-continuous-scan-config
export ECR_SCAN_RESULT_BUCKET=ecr-continuous-scan-result

aws s3api create-bucket \
            --bucket $ECR_SCAN_SVC_BUCKET \
            --create-bucket-configuration LocationConstraint=$(aws configure get region) \
            --region $(aws configure get region)

aws s3api create-bucket \
            --bucket $ECR_SCAN_CONFIG_BUCKET \
            --create-bucket-configuration LocationConstraint=$(aws configure get region) \
            --region $(aws configure get region)

aws s3api create-bucket \
            --bucket $ECR_SCAN_RESULT_BUCKET \
            --create-bucket-configuration LocationConstraint=$(aws configure get region) \
            --region $(aws configure get region)
```

Interaction:

```
export ECRSCANAPI_URL=$(aws cloudformation describe-stacks --stack-name ecr-continuous-scan | jq '.Stacks[].Outputs[] | select(.OutputKey=="ECRScanAPIEndpoint").OutputValue' -r)
curl -s --header "Content-Type: application/json" --request POST --data @test-scan-spec.json $ECRSCANAPI_URL/register/
curl $ECRSCANAPI_URL/summary
```