package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/aws"

	uuid "github.com/satori/go.uuid"
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

// rmClusterSpec deletes the scan spec in a given bucket
func rmClusterSpec(configbucket, scanid string) error {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return err
	}
	svc := s3.New(cfg)
	req := svc.DeleteObjectRequest(&s3.DeleteObjectInput{
		Bucket: aws.String(configbucket),
		Key:    aws.String(scanid + ".json"),
	})
	_, err = req.Send(context.Background())
	if err != nil {
		return err
	}
	return nil
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	configbucket := os.Getenv("ECR_SCAN_CONFIG_BUCKET")
	fmt.Printf("DEBUG:: config continuous scan start\n")

	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return serverError(err)
	}
	svc := s3.New(cfg)

	switch request.HTTPMethod {
	case "POST":
		fmt.Printf("DEBUG:: adding scan config\n")
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
		err = storeScanSpec(configbucket, ss)
		if err != nil {
			return serverError(err)
		}
		msg := fmt.Sprintf("Added scan config. ID=%v ", ss.ID)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusOK,
			Headers: map[string]string{
				"Content-Type":                "application/json",
				"Access-Control-Allow-Origin": "*",
			},
			Body: msg,
		}, nil
	case "DELETE":
		fmt.Printf("DEBUG:: removing scan config\n")
		// validate repo in URL path:
		if _, ok := request.PathParameters["id"]; !ok {
			return serverError(fmt.Errorf("Unknown configuration"))
		}
		req := svc.ListObjectsRequest(&s3.ListObjectsInput{
			Bucket: &configbucket,
		},
		)
		resp, err := req.Send(context.TODO())
		if err != nil {
			return serverError(err)
		}
		for _, obj := range resp.Contents {
			fn := *obj.Key
			scanID := strings.TrimSuffix(fn, ".json")
			if scanID == request.PathParameters["id"] {
				rmClusterSpec(configbucket, scanID)
				msg := fmt.Sprintf("Deleted scan config %v ", request.PathParameters["id"])
				return events.APIGatewayProxyResponse{
					StatusCode: http.StatusOK,
					Headers: map[string]string{
						"Content-Type":                "application/json",
						"Access-Control-Allow-Origin": "*",
					},
					Body: msg,
				}, nil
			}
		}
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusNotFound,
			Headers: map[string]string{
				"Content-Type":                "application/json",
				"Access-Control-Allow-Origin": "*",
			},
			Body: "This scan config does not exist, no operation performed",
		}, nil
	case "GET":
		fmt.Printf("DEBUG:: listing scan config\n")
		req := svc.ListObjectsRequest(&s3.ListObjectsInput{
			Bucket: &configbucket,
		},
		)
		resp, err := req.Send(context.TODO())
		if err != nil {
			return serverError(err)
		}
		scanspecs := []ScanSpec{}
		for _, obj := range resp.Contents {
			fn := *obj.Key
			scanID := strings.TrimSuffix(fn, ".json")
			scanspec, err := fetchScanSpec(configbucket, scanID)
			if err != nil {
				return serverError(err)
			}
			scanspecs = append(scanspecs, scanspec)

		}
		scanspecsjson, err := json.Marshal(scanspecs)
		if err != nil {
			return serverError(err)
		}
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusOK,
			Headers: map[string]string{
				"Content-Type":                "application/json",
				"Access-Control-Allow-Origin": "*",
			},
			Body: string(scanspecsjson),
		}, nil
	}
	fmt.Printf("DEBUG:: register continuous scan done\n")
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusMethodNotAllowed,
		Headers: map[string]string{
			"Access-Control-Allow-Origin": "*",
		},
	}, nil
}

func main() {
	lambda.Start(handler)
}
