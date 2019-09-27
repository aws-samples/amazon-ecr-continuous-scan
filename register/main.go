package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/aws"
	uuid "github.com/satori/go.uuid"
)

// ScanSpec represents configuration for the
// target repository
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

// storeScanSpec stores the scan spec in a given bucket
func storeScanSpec(configbucket string, scanspec ScanSpec) error {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return err
	}
	ssjson, err := json.Marshal(scanspec)
	if err != nil {
		return err
	}
	uploader := s3manager.NewUploader(cfg)
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(configbucket),
		Key:    aws.String(scanspec.ID + ".json"),
		Body:   strings.NewReader(string(ssjson)),
	})
	return err
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	configbucket := os.Getenv("ECR_SCAN_CONFIG_BUCKET")
	fmt.Printf("DEBUG:: register continuous scan start\n")
	fmt.Println(configbucket)
	ss := ScanSpec{}
	// Unmarshal the JSON payload in the POST:
	err := json.Unmarshal([]byte(request.Body), &ss)
	if err != nil {
		return serverError(err)
	}
	specID, err := uuid.NewV4()
	if err != nil {
		return serverError(err)
	}
	ss.ID = specID.String()
	ss.CreationTime = fmt.Sprintf("%v", time.Now().Unix())
	storeScanSpec(configbucket, ss)
	fmt.Printf("DEBUG:: register continuous scan done\n")
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*",
		},
		Body: ss.ID,
	}, nil
}

func main() {
	lambda.Start(handler)
}
