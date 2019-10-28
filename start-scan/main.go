package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

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
	// Tags to take into consideration, if empty, all tags will be scanned
	Tags []string `json:"tags"`
}

func startScan(scanspec ScanSpec) error {
	s := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(scanspec.Region),
	}))
	svc := ecr.New(s)
	scaninput := &ecr.StartImageScanInput{
		RepositoryName: &scanspec.Repository,
		RegistryId:     &scanspec.RegistryID,
	}
	switch len(scanspec.Tags) {
	case 0: // empty list of tags, scan all tags:
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
			return err
		}
		for _, iid := range lio.ImageIds {
			scaninput.ImageId = iid
			result, err := svc.StartImageScan(scaninput)
			if err != nil {
				fmt.Println(err)
				return err
			}
			fmt.Printf("DEBUG:: result for tag %v: %v\n", *iid.ImageTag, result)
		}

	default: // iterate over the tags specified in the config:
		fmt.Printf("DEBUG:: scanning tags %v for repo %v\n", scanspec.Tags, scanspec.Repository)
		for _, tag := range scanspec.Tags {
			scaninput.ImageId = &ecr.ImageIdentifier{
				ImageTag: aws.String(tag),
			}
			result, err := svc.StartImageScan(scaninput)
			if err != nil {
				fmt.Println(err)
				return err
			}
			fmt.Printf("DEBUG:: result for tag %v: %v\n", tag, result)
		}
	}
	return nil
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

func handler() error {
	configbucket := os.Getenv("ECR_SCAN_CONFIG_BUCKET")
	fmt.Printf("DEBUG:: scan start\n")
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		fmt.Println(err)
		return err
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
		return err
	}
	for _, obj := range resp.Contents {
		fn := *obj.Key
		scanID := strings.TrimSuffix(fn, ".json")
		scanspec, err := fetchScanSpec(configbucket, scanID)
		if err != nil {
			fmt.Println(err)
			return err
		}
		err = startScan(scanspec)
		if err != nil {
			fmt.Println(err)
			return err
		}
	}
	fmt.Printf("DEBUG:: scan done\n")
	return nil
}

func main() {
	lambda.Start(handler)
}
