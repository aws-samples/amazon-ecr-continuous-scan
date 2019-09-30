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
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/s3manager"
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
	// Level specifies the severity to consider for summaries
	// 'high' ... HIGH only, and 'all' ... INFORMATIONAL+UNDEFINED+LOW+MEDIUM+HIGH
	Level string `json:"level"`
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
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return ss, err
	}
	downloader := s3manager.NewDownloader(cfg)
	buf := aws.NewWriteAtBuffer([]byte{})
	_, err = downloader.Download(buf, &s3.GetObjectInput{
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

func describeScan(scanspec ScanSpec) (string, error) {
	s := session.Must(session.NewSession(&aws.Config{
		Region:   aws.String(scanspec.Region),
		Endpoint: aws.String("https://starport.us-west-2.amazonaws.com"),
	}))
	svc := ecr.New(s)
	iid := &ecr.ImageIdentifier{
		ImageTag: aws.String("latest"),
	}
	input := &ecr.DescribeImageScanFindingsInput{
		RepositoryName: &scanspec.Repository,
		RegistryId:     &scanspec.RegistryID,
		ImageId:        iid,
		MaxResults:     aws.Int64(10),
	}
	result, err := svc.DescribeImageScanFindings(input)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%v", result), nil
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	configbucket := os.Getenv("ECR_SCAN_CONFIG_BUCKET")
	fmt.Printf("DEBUG:: summary start\n")
	result := ""
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		fmt.Println(err)
		return serverError(err)
	}
	svc := s3.New(cfg)
	fmt.Printf("Scanning bucket %v for scan specs\n", configbucket)
	req := svc.ListObjectsRequest(&s3.ListObjectsInput{
		Bucket: &configbucket,
	},
	)
	resp, err := req.Send(context.TODO())
	if err != nil {
		fmt.Println(err)
		return serverError(err)
	}
	for _, obj := range resp.Contents {
		fn := *obj.Key
		scanID := strings.TrimSuffix(fn, ".json")
		scanspec, err := fetchScanSpec(configbucket, scanID)
		if err != nil {
			fmt.Println(err)
			return serverError(err)
		}
		result, err = describeScan(scanspec)
		if err != nil {
			fmt.Println(err)
			return serverError(err)
		}
		fmt.Printf("DEBUG:: result %v\n", result)
	}

	fmt.Printf("DEBUG:: summary done\n")
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*",
		},
		Body: result,
	}, nil
}

func main() {
	lambda.Start(handler)
}
