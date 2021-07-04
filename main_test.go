package main

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

var event_applied = "test-events/event_applied.json"
var event_errored = "test-events/event_errored.json"
var event_not_modified = "test-events/event_not_modified.json"
var key string

func init() {
	bucket = os.Getenv("BUCKET")
}

func TestStatusModifiedApplied(t *testing.T) {
	eventFile, _ := ioutil.ReadFile(event_applied)
	payload := Payload{}
	event := events.APIGatewayProxyRequest{}
	err := json.Unmarshal([]byte(eventFile), &event)
	if err != nil {
		log.Println("Error loading", event_applied, err)
	}
	err = json.Unmarshal([]byte(event.Body), &payload)
	if err != nil {
		log.Println(err)
	}
	statusModified := StatusModified(payload)
	assert.True(t, statusModified, "status = applied")
}

func TestStatusModifiedErrored(t *testing.T) {
	eventFile, _ := ioutil.ReadFile(event_errored)
	payload := Payload{}
	event := events.APIGatewayProxyRequest{}
	err := json.Unmarshal([]byte(eventFile), &event)
	if err != nil {
		log.Println("Error loading", event_errored, err)
	}
	err = json.Unmarshal([]byte(event.Body), &payload)
	if err != nil {
		log.Println(err)
	}
	statusModified := StatusModified(payload)
	assert.True(t, statusModified, "status = errored")
}

func TestStatusModifiedFalse(t *testing.T) {
	eventFile, _ := ioutil.ReadFile(event_not_modified)
	payload := Payload{}
	event := events.APIGatewayProxyRequest{}
	err := json.Unmarshal([]byte(eventFile), &event)
	if err != nil {
		log.Println("Error loading", event_not_modified, err)
	}
	err = json.Unmarshal([]byte(event.Body), &payload)
	if err != nil {
		log.Println(err)
	}
	statusModified := StatusModified(payload)
	assert.False(t, statusModified, "status = canceled")
}

func TestGetLastStateVersion(t *testing.T) {
	eventFile, _ := ioutil.ReadFile(event_applied)
	payload := Payload{}
	event := events.APIGatewayProxyRequest{}
	err := json.Unmarshal([]byte(eventFile), &event)
	if err != nil {
		log.Println("Error loading", event_applied, err)
	}
	err = json.Unmarshal([]byte(event.Body), &payload)
	if err != nil {
		log.Println(err)
	}

	_, err = GetLastStateVersion(payload.WorkspaceID)
	if err != nil {
		t.Fail()
	}
}

func TestDownloadBody(t *testing.T) {
	eventFile, _ := ioutil.ReadFile(event_applied)
	payload := Payload{}
	event := events.APIGatewayProxyRequest{}
	err := json.Unmarshal([]byte(eventFile), &event)
	if err != nil {
		log.Println("Error loading", event_applied, err)
	}
	err = json.Unmarshal([]byte(event.Body), &payload)
	if err != nil {
		log.Println(err)
	}

	state, err := GetLastStateVersion(payload.WorkspaceID)
	if err != nil {
		t.Fail()
	}

	_, err = DownloadBody(state.DownloadURL)

	if err != nil {
		t.Fail()
	}

}

func TestUploadToS3(t *testing.T) {
	err := UploadToS3("test", "test")
	if err != nil {
		log.Println(err)
		t.Fail()
	}
	err = deleteFromS3("test")
	if err != nil {
		log.Println(err)
		t.Fail()
	}
}

func TestSaveLastState(t *testing.T) {
	eventFile, _ := ioutil.ReadFile(event_applied)
	payload := Payload{}
	event := events.APIGatewayProxyRequest{}
	err := json.Unmarshal([]byte(eventFile), &event)
	if err != nil {
		log.Println("Error loading", event_applied, err)
	}
	err = json.Unmarshal([]byte(event.Body), &payload)
	if err != nil {
		log.Println(err)
	}

	key, err = SaveLastState(payload)
	if err != nil {
		log.Println(err)
		t.Fail()
	}
	err = deleteFromS3(key)
	if err != nil {
		log.Println(err)
		t.Fail()
	}
}

func TestHandler(t *testing.T) {

	eventFile, _ := ioutil.ReadFile(event_applied)
	event := events.APIGatewayProxyRequest{}
	err := json.Unmarshal([]byte(eventFile), &event)
	if err != nil {
		log.Println("Error loading", event_applied, err)
	}

	state, err := Handler(context.Background(), event)

	if err != nil {
		log.Println(err, state)
		t.Fail()
	}
	err = deleteFromS3(key)
	if err != nil {
		log.Println(err)
		t.Fail()
	}
	assert.Equal(t, 200, state.StatusCode)

}

func deleteFromS3(key string) error {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	client := s3.NewFromConfig(cfg)
	_, err = client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	return err
}
