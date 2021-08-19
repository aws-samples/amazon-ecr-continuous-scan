package main

import (
	"context"
	"encoding/json"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
)

// ScanSpec represents configuration for the target repository
type ScanSpec struct {
	// ID is a unique identifier for the scan spec
	ID string `json:"id"`
	// CreationTime is the UTC timestamp of when the scan spec was created
	CreationTime string `json:"created"`
	// Region specifies the region the repository is in
	Region string `json:"region"`
	// RegistryID specifies the registry ID
	RegistryID string `json:"registry"`
	// Repository specifies the repository name
	Repository string `json:"repository"`
	// Tags to take into consideration, if empty, all tags will be scanned
	Tags []string `json:"tags"`
}

func serverError(err error) (events.APIGatewayProxyResponse, error) {
	fmt.Println(err.Error())
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusInternalServerError,
		Headers: map[string]string{
			"Access-Control-Allow-Origin": "*",
		},
		Body: fmt.Sprintf("%v", err.Error()),
	}, nil
}

// fetchScanSpec returns the scan spec
// in a given bucket, with a given scan ID
func fetchScanSpec(configbucket, scanid string) (ScanSpec, error) {
	ss := ScanSpec{}
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return ss, err
	}

	// Create an S3 Client with the config
	client := s3.NewFromConfig(cfg)

	// Create an uploader passing it the client
	downloader := manager.NewDownloader(client)

	buf := aws.NewWriteAtBuffer([]byte{})
	_, err = downloader.Download(context.TODO(), buf, &s3.GetObjectInput{
		Bucket: aws.String(configbucket),
		Key:    aws.String(scanid + ".json"),
	})
	if err != nil {
		return ss, err
	}
	err = json.Unmarshal(buf.Bytes(), &ss)
	if err != nil {
		return ss, err
	}
	return ss, nil
}

func describeScan(scanspec ScanSpec) (map[string]ecr.ImageScanFindings, error) {
	s := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(scanspec.Region),
	}))
	svc := ecr.New(s)
	descinput := &ecr.DescribeImageScanFindingsInput{
		RepositoryName: &scanspec.Repository,
		RegistryId:     &scanspec.RegistryID,
	}
	results := map[string]ecr.ImageScanFindings{}
	switch len(scanspec.Tags) {
	case 0: // empty list of tags, describe all tags:
		fmt.Printf("DEBUG:: scanning all tags for repo %v\n", scanspec.Repository)
		lio, err := svc.ListImages(&ecr.ListImagesInput{
			RepositoryName: &scanspec.Repository,
			RegistryId:     &scanspec.RegistryID,
			Filter: &ecr.ListImagesFilter{
				TagStatus: aws.String("TAGGED"),
			},
		})
		if err != nil {
			fmt.Println(err)
			return results, err
		}
		for _, iid := range lio.ImageIds {
			descinput.ImageId = iid
			result, err := svc.DescribeImageScanFindings(descinput)
			if err != nil {
				return results, err
			}
			results[*iid.ImageTag] = *result.ImageScanFindings
			// fmt.Printf("DEBUG:: result for tag %v: %v\n", *iid.ImageTag, result)
		}
	default: // iterate over the tags specified in the config:
		fmt.Printf("DEBUG:: scanning tags %v for repo %v\n", scanspec.Tags, scanspec.Repository)
		for _, tag := range scanspec.Tags {
			descinput.ImageId = &ecr.ImageIdentifier{
				ImageTag: aws.String(tag),
			}
			result, err := svc.DescribeImageScanFindings(descinput)
			if err != nil {
				fmt.Println(err)
				return results, err
			}
			results[tag] = *result.ImageScanFindings
			// fmt.Printf("DEBUG:: result for tag %v: %v\n", tag, result)
		}
	}
	return results, nil
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	configbucket := os.Getenv("ECR_SCAN_CONFIG_BUCKET")
	fmt.Printf("DEBUG:: summary start\n")
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		fmt.Println(err)
		return serverError(err)
	}
	svc := s3.NewFromConfig(cfg)
	fmt.Printf("Scanning bucket %v for scan specs\n", configbucket)
	resp, err := svc.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: &configbucket,
	},
	)
	// resp, err := req.Send(context.TODO())
	if err != nil {
		fmt.Println(err)
		return serverError(err)
	}
	ssresult := ""
	for _, obj := range resp.Contents {
		fn := *obj.Key
		scanID := strings.TrimSuffix(fn, ".json")
		scanspec, err := fetchScanSpec(configbucket, scanID)
		if err != nil {
			fmt.Println(err)
			return serverError(err)
		}
		results, err := describeScan(scanspec)
		if err != nil {
			fmt.Println(err)
			return serverError(err)
		}
		for tag, result := range results {
			sevcount := ""
			for sev, count := range result.FindingSeverityCounts {
				sevcount += fmt.Sprintf(" %v: %v\n", sev, *count)
			}
			ssresult += fmt.Sprintf("Results for %v:%v in %v:\n%v\n\n", scanspec.Repository, tag, scanspec.Region, sevcount)
		}
	}

	fmt.Printf("DEBUG:: summary done\n")
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*",
		},
		Body: ssresult,
	}, nil
}

func main() {
	lambda.Start(handler)
}
