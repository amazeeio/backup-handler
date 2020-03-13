package handler

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/amazeeio/lagoon-cli/pkg/api"
)

func checkEqual(t *testing.T, got, want interface{}, msgs ...interface{}) {
	if !reflect.DeepEqual(got, want) {
		buf := bytes.Buffer{}
		buf.WriteString("got:\n[%v]\nwant:\n[%v]\n")
		for _, v := range msgs {
			buf.WriteString(v.(string))
		}
		t.Errorf(buf.String(), got, want)
	}
}

func TestProcessBackups(t *testing.T) {
	broker := RabbitBroker{
		Hostname:     os.Getenv("BROKER_ADDRESS"),
		Username:     os.Getenv("BROKER_USER"),
		Password:     os.Getenv("BROKER_PASS"),
		Port:         os.Getenv("BROKER_PORT"),
		QueueName:    "lagoon-webhooks:queue",
		ExchangeName: "lagoon-webhooks",
	}
	graphQL := GraphQLEndpoint{
		Endpoint:        os.Getenv("GRAPHQL_ENDPOINT"),
		TokenSigningKey: os.Getenv("JWT_SECRET"),
		JWTAudience:     os.Getenv("JWT_AUDIENCE"),
	}
	backupHandler, err := NewBackupHandler(broker, graphQL)
	if err != nil {
		t.Errorf("unable to create backuphandler, error is %s:", err)
	}
	var backupData Backups
	jsonBackupTestData, err := ioutil.ReadFile("testdata/example-com.json")
	if err != nil {
		t.Errorf("unable to read file, error is %s:", err.Error())
	}
	resultTestData, err := ioutil.ReadFile("testdata/example-com.result")
	if err != nil {
		t.Errorf("unable to read file, error is %s:", err.Error())
	}
	decoder := json.NewDecoder(bytes.NewReader(jsonBackupTestData))
	err = decoder.Decode(&backupData)
	if err != nil {
		t.Errorf("unable to decode json, error is %s:", err.Error())
	}
	var backupsEnv api.Environment
	addBackups := backupHandler.ProcessBackups(backupData, backupsEnv)
	var backupResult []string
	for _, backup := range addBackups {
		backupResult = append(backupResult, backup.Body.Snapshots[0].Hostname)
	}
	bResult := strings.Join(backupResult, ",")
	if string(bResult) != string(resultTestData) {
		checkEqual(t, string(bResult), string(resultTestData), "processing failed")
	}
}
