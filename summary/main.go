package main

import (
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// ClusterSpec represents the parameters for eksctl,
// as cluster metadata including owner and how long the cluster
// still has to live.
type ClusterSpec struct {
	// ID is a unique identifier for the cluster
	ID string `json:"id"`
	// Name specifies the cluster name
	Name string `json:"name"`
	// NumWorkers specifies the number of worker nodes, defaults to 1
	NumWorkers int `json:"numworkers"`
	// KubeVersion  specifies the Kubernetes version to use, defaults to `1.12`
	KubeVersion string `json:"kubeversion"`
	// Timeout specifies the timeout in minutes, after which the cluster
	// is destroyed, defaults to 10
	Timeout int `json:"timeout"`
	// Timeout specifies the cluster time to live in minutes.
	// In other words: the remaining time the cluster has before it is destroyed
	TTL int `json:"ttl"`
	// Owner specifies the email address of the owner (will be notified when cluster is created and 5 min before destruction)
	Owner string `json:"owner"`
	// CreationTime is the UTC timestamp of when the cluster was created
	// which equals the point in time of the creation of the respective
	// JSON representation of the cluster spec as an object in the metadata
	// bucket
	CreationTime string `json:"created"`
	// ClusterDetails is only valid for lookup of individual clusters,
	// that is, when user does, for example, a eksp l CLUSTERID. It
	// holds info such as cluster status and config
	ClusterDetails map[string]string `json:"details"`
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

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	configbucket := os.Getenv("ECR_SCAN_CONFIG_BUCKET")
	resultbucket := os.Getenv("ECR_SCAN_RESULT_BUCKET")
	fmt.Printf("DEBUG:: summary start\n")
	fmt.Println(configbucket)
	fmt.Println(resultbucket)
	// cfg, err := external.LoadDefaultAWSConfig()
	// if err != nil {
	// 	return serverError(err)
	// }
	// // validate cluster ID or list lookup in URL path:
	// if _, ok := request.PathParameters["clusterid"]; !ok {
	// 	return serverError(fmt.Errorf("Unknown cluster status query. Either specify a cluster ID or _ for listing all clusters."))
	// }
	// cID := request.PathParameters["clusterid"]
	// // return info on specified cluster if we have an cluster ID in the URL path component:
	// if cID != "*" {
	// 	fmt.Printf("DEBUG:: cluster info lookup for ID %v start\n", cID)
	// 	cs, err := fetchClusterSpec(clusterbucket, cID)
	// 	if err != nil {
	// 		return serverError(err)
	// 	}
	// 	clustername := cs.Name
	// 	cd, err := getClusterDetails(clustername)
	// 	if err != nil {
	// 		return serverError(err)
	// 	}
	// 	cs.ClusterDetails = make(map[string]string)
	// 	cs.ClusterDetails["endpoint"] = *cd.Endpoint
	// 	cs.ClusterDetails["status"] = fmt.Sprintf("%v", cd.Status)
	// 	cs.ClusterDetails["platformv"] = *cd.PlatformVersion
	// 	cs.ClusterDetails["vpcconf"] = fmt.Sprintf("private access: %v, public access: %v ", *cd.ResourcesVpcConfig.EndpointPrivateAccess, *cd.ResourcesVpcConfig.EndpointPublicAccess)
	// 	cs.ClusterDetails["iamrole"] = *cd.RoleArn
	// 	csjson, err := json.Marshal(cs)
	// 	if err != nil {
	// 		return serverError(err)
	// 	}
	// 	fmt.Printf("DEBUG:: cluster info lookup done\n")
	// 	return events.APIGatewayProxyResponse{
	// 		StatusCode: http.StatusOK,
	// 		Headers: map[string]string{
	// 			"Content-Type":                "application/json",
	// 			"Access-Control-Allow-Origin": "*",
	// 		},
	// 		Body: string(csjson),
	// 	}, nil
	// }
	// // if we have no specified cluster ID in the path, list all cluster IDs:
	// // list all objects in the bucket:
	// svc := s3.New(cfg)
	// req := svc.ListObjectsRequest(&s3.ListObjectsInput{Bucket: &clusterbucket})
	// resp, err := req.Send(context.TODO())
	// if err != nil {
	// 	return serverError(err)
	// }
	// clusterIDs := []string{}
	// // get the content of all objects (cluster specs) in the bucket:
	// for _, obj := range resp.Contents {
	// 	fn := *obj.Key
	// 	clusterIDs = append(clusterIDs, strings.TrimSuffix(fn, ".json"))
	// }
	// clusteridsjson, err := json.Marshal(clusterIDs)
	// if err != nil {
	// 	return serverError(err)
	// }

	fmt.Printf("DEBUG:: summary done\n")
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*",
		},
		Body: "summary done",
	}, nil
}

func main() {
	lambda.Start(handler)
}
