package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/hashicorp/go-tfe"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"strings"
	"time"
)

type Payload struct {
	PayloadVersion              int       `json:"payload_version"`
	NotificationConfigurationID string    `json:"notification_configuration_id"`
	RunURL                      string    `json:"run_url"`
	RunID                       string    `json:"run_id"`
	RunMessage                  string    `json:"run_message"`
	RunCreatedAt                time.Time `json:"run_created_at"`
	RunCreatedBy                string    `json:"run_created_by"`
	WorkspaceID                 string    `json:"workspace_id"`
	WorkspaceName               string    `json:"workspace_name"`
	OrganizationName            string    `json:"organization_name"`
	Notifications               []struct {
		Message      string    `json:"message"`
		Trigger      string    `json:"trigger"`
		RunStatus    string    `json:"run_status"`
		RunUpdatedAt time.Time `json:"run_updated_at"`
		RunUpdatedBy string    `json:"run_updated_by"`
	} `json:"notifications"`
}

var tfe_token string = os.Getenv("TF_TOKEN")
var bucket string = os.Getenv("BUCKET")

func UploadToS3(key string, body string) error {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	client := s3.NewFromConfig(cfg)
	uploader := manager.NewUploader(client)
	_, err = uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader([]byte(body)),
	})
	return err
}

func DownloadBody(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	buff := new(bytes.Buffer)
	_, err = buff.ReadFrom(resp.Body)
	if err != nil {
		return "", err
	}
	return buff.String(), err
}

func GetLastStateVersion(workspaceId string) (*tfe.StateVersion, error) {
	var state *tfe.StateVersion

	tfe_client, err := tfe.NewClient(&tfe.Config{
		Token: tfe_token,
	})
	if err != nil {
		log.Println(err)
		return state, err
	}
	state, err = tfe_client.StateVersions.Current(context.Background(), workspaceId)
	if err != nil {
		log.Println("Error", err)
		return state, err
	}
	return state, nil
}

func SaveLastState(payload Payload) (string, error) {
	state, err := GetLastStateVersion(payload.WorkspaceID)
	if err != nil {
		return "", err
	}

	key := fmt.Sprintf("%s/%s/%s/%s-%v-%s-%s.json", payload.OrganizationName, payload.WorkspaceName, state.CreatedAt.Format("2006/01/02"),
		payload.WorkspaceName, state.Serial, state.ID, payload.RunID)

	body, err := DownloadBody(state.DownloadURL)
	if err != nil {
		return "", err
	}
	err = UploadToS3(key, body)
	if err == nil {
		log.Println("Terraform State saved in: ", key)
	}
	return key, err
}

func StatusModified(payload Payload) bool {
	for _, v := range payload.Notifications {
		return strings.Contains(v.RunStatus, "errored") || strings.Contains(v.RunStatus, "applied")
	}
	return false
}

func Handler(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	payload := Payload{}
	json.Unmarshal([]byte(event.Body), &payload)

	if StatusModified(payload) {
		_, err := SaveLastState(payload)
		if err != nil {
			log.Println(err)
			return events.APIGatewayProxyResponse{StatusCode: 500}, err
		}
	}
	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

func main() {
	lambda.Start(Handler)
}
