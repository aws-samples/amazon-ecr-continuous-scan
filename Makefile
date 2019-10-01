ECR_SCAN_STACK_NAME?=ecr-continuous-scan
ECR_SCAN_SVC_BUCKET?=ecr-continuous-scan-svc
ECR_SCAN_CONFIG_BUCKET?=ecr-continuous-scan-config

.PHONY: build up deploy destroy status


build: bconfigs bsscan bsummary bfindings

bconfigs:
	GOOS=linux GOARCH=amd64 go build -v -ldflags '-d -s -w' -a -tags netgo -installsuffix netgo -o bin/configs ./configs

bsscan:
	GOOS=linux GOARCH=amd64 go build -v -ldflags '-d -s -w' -a -tags netgo -installsuffix netgo -o bin/start-scan ./start-scan

bsummary:
	GOOS=linux GOARCH=amd64 go build -v -ldflags '-d -s -w' -a -tags netgo -installsuffix netgo -o bin/summary ./summary

bfindings:
	GOOS=linux GOARCH=amd64 go build -v -ldflags '-d -s -w' -a -tags netgo -installsuffix netgo -o bin/findings ./findings

up: 
	sam package --template-file template.yaml --output-template-file current-stack.yaml --s3-bucket ${ECR_SCAN_SVC_BUCKET}
	sam deploy --template-file current-stack.yaml --stack-name ${ECR_SCAN_STACK_NAME} --capabilities CAPABILITY_IAM --parameter-overrides ConfigBucketName="${ECR_SCAN_CONFIG_BUCKET}"

deploy: build up

destroy:
	aws cloudformation delete-stack --stack-name ${ECR_SCAN_STACK_NAME}

status:
	aws cloudformation describe-stacks --stack-name ${ECR_SCAN_STACK_NAME}