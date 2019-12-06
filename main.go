package main

/*
	this is a simple wrestic snapshot webhook handler for lagoon

	https://nesv.github.io/golang/2014/02/25/worker-queues-in-go.html
	workerqueues maybe in the event rabbit goes away
*/

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/amazeeio/lagoon-cli/api"
	"github.com/google/uuid"
	"github.com/isayme/go-amqp-reconnect/rabbitmq"
	"github.com/streadway/amqp"
)

type webhook struct {
	Webhooktype string  `json:"webhooktype"`
	Event       string  `json:"event"`
	UUID        string  `json:"uuid"`
	Body        backups `json:"body"`
}

type backups struct {
	Name            string        `json:"name"`
	BucketName      string        `json:"bucket_name"`
	BackupMetrics   backupMetrics `json:"backup_metrics"`
	Snapshots       []snapshot    `json:"snapshots"`
	RestoreLocation string        `json:"restore_location"`
	SnapshotID      string        `json:"snapshot_ID"`
	RestoredFiles   []string      `json:"restored_files"`
}

type backupMetrics struct {
	BackupStartTimestamp int         `json:"backup_start_timestamp"`
	BackupEndTimestamp   int         `json:"backup_end_timestamp"`
	Errors               int         `json:"errors"`
	NewFiles             int         `json:"new_files"`
	ChangedFiles         int         `json:"changed_files"`
	UnmodifiedFiles      int         `json:"unmodified_files"`
	NewDirs              int         `json:"new_dirs"`
	ChangedDirs          int         `json:"changed_dirs"`
	UnmodifiedDirs       int         `json:"unmodified_dirs"`
	DataTransferred      int         `json:"data_transferred"`
	MountedPVCs          interface{} `json:"mounted_PVCs"`
	Folder               string      `json:"Folder"`
}

type snapshot struct {
	ID       string      `json:"id"`
	Time     time.Time   `json:"time"`
	Tree     string      `json:"tree"`
	Paths    []string    `json:"paths"`
	Hostname string      `json:"hostname"`
	Username string      `json:"username"`
	UID      int         `json:"uid"`
	Gid      int         `json:"gid"`
	Tags     interface{} `json:"tags"`
}

var (
	// rabbit connection info
	rabbitConn    *rabbitmq.Connection
	rabbitChannel *rabbitmq.Channel
	queueName     = "lagoon-webhooks:queue"
	exchangeName  = "lagoon-webhooks"

	// get these from envvars, but fail if they don't exist
	brokerAddress = os.Getenv("BROKER_ADDRESS")
	brokerPort    = os.Getenv("BROKER_PORT")
	brokerUser    = os.Getenv("BROKER_USER")
	brokerPass    = os.Getenv("BROKER_PASS")
	// we can generate a token with these using the lagoon-api package
	tokenSigningKey = os.Getenv("JWT_SECRET")
	jwtAudience     = os.Getenv("JWT_AUDIENCE")
	graphQLEndpoint = os.Getenv("GRAPHQL_ENDPOINT")
	httpListenPort  = os.Getenv("HTTP_LISTEN_PORT")

	// set up the rabbit endpoint
	amqpURI = "amqp://" + brokerUser + ":" + brokerPass + "@" + brokerAddress + ":" + brokerPort
)

func init() {
	initAmqp()
}

func initAmqp() {
	// github.com/isayme/go-amqp-reconnect/rabbitmq
	// reconnect to rabbit automatically eventually, but still accept webhooks (just fails and webhook data is lost)
	var err error
	rabbitConn, err = rabbitmq.Dial(amqpURI)
	failOnError(err, "Failed to connect to RabbitMQ")
	rabbitChannel, err = rabbitConn.Channel()
	failOnError(err, "Failed to open a channel")
	err = rabbitChannel.ExchangeDeclare(
		exchangeName, // name
		"direct",     // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	failOnError(err, "Could not declare exchange")
	queue, err := rabbitChannel.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil)
	failOnError(err, "Could not declare queue")
	err = rabbitChannel.QueueBind(
		queue.Name,   // queue name
		"",           // routing key
		exchangeName, // exchange
		false,
		nil)
	failOnError(err, "Failed to bind queue")
}

func main() {
	// we want all of these vars, else fail
	if len(os.Getenv("BROKER_ADDRESS")) == 0 {
		log.Fatalln("BROKER_ADDRESS not set")
	}
	if len(os.Getenv("BROKER_PORT")) == 0 {
		log.Fatalln("BROKER_PORT not set")
	}
	if len(os.Getenv("BROKER_USER")) == 0 {
		log.Fatalln("BROKER_USER not set")
	}
	if len(os.Getenv("BROKER_PASS")) == 0 {
		log.Fatalln("BROKER_PASS not set")
	}
	if len(os.Getenv("JWT_SECRET")) == 0 {
		log.Fatalln("JWT_SECRET not set")
	}
	if len(os.Getenv("JWT_AUDIENCE")) == 0 {
		log.Fatalln("JWT_AUDIENCE not set")
	}
	if len(os.Getenv("GRAPHQL_ENDPOINT")) == 0 {
		log.Fatalln("GRAPHQL_ENDPOINT not set")
	}
	if len(os.Getenv("HTTP_LISTEN_PORT")) == 0 {
		httpListenPort = "3000"
	}

	// handle the requests
	http.HandleFunc("/", webhookHandler)
	http.ListenAndServe(":"+httpListenPort, nil)

}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	var backupData backups
	// decode the body result into the backups struct
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&backupData)
	if err != nil {
		log.Printf("unable to handle webhook, error is %s:", err.Error())
	} else {
		// get backups from the API
		lagoonAPI, err := api.New(tokenSigningKey, jwtAudience, graphQLEndpoint)
		if err != nil {
			log.Printf("unable to handle webhook, error is %s:", err.Error())
			return
		}

		// handle restores
		if backupData.RestoreLocation != "" {
			singleBackup := webhook{
				Webhooktype: "resticbackup",
				Event:       "restore:finished",
				UUID:        uuid.New().String(),
				Body:        backupData,
			}
			addToMessageQueue(singleBackup)
			// else handle snapshots
		} else if backupData.Snapshots != nil {
			// use the name from the webhook to get the environment in the api
			environment := api.EnvironmentBackups{
				OpenshiftProjectName: backupData.Name,
			}
			envBackups, err := lagoonAPI.GetEnvironmentBackups(environment)
			if err != nil {
				log.Printf("unable to get backups from api, error is %s:", err.Error())
				return
			}
			// unmarshal the result into the environment struct
			var backupsEnv api.Environment
			json.Unmarshal(envBackups, &backupsEnv)
			// remove backups that no longer exists from the api
			for index, backup := range backupsEnv.Backups {
				// check that the backup in the api is not in the webhook payload
				if !apiBackupInWebhook(backupData.Snapshots, backup.BackupID) {
					// if the backup in the api is not in the webhook payload
					// remove it from the webhook payload data
					removeSnapshot(backupData.Snapshots, index)
					delBackup := api.DeleteBackup{
						BackupID: backup.BackupID,
					}
					// now delete it from the api as it no longer exists
					_, err := lagoonAPI.DeleteBackup(delBackup) // result is always success, or will error
					if err != nil {
						log.Printf("unable to delete backup from api, error is %s:", err.Error())
						return
					}
					log.Printf("deleted backup %s for %s", backup.BackupID, backupsEnv.OpenshiftProjectName)
				}
			}

			// if we get this far, then the payload data from the webhook should only have snapshots that are new or exist in the api
			for _, snapshotData := range backupData.Snapshots {
				// we want to check that we match the name to the project/environment properly and capture any prebackuppods too
				matched, _ := regexp.MatchString("^"+backupData.Name+"-.*-prebackuppod$|^"+backupData.Name+"$", snapshotData.Hostname)
				if matched {
					// if the snapshot id is not in already in the api, then we want to add this backup to the webhooks queue
					// this results in far less messages being sent to the queue as only new snapshots will be added
					if !backupInEnvironment(backupsEnv, snapshotData.ID) {
						singleBackup := webhook{
							Webhooktype: "resticbackup",
							Event:       "snapshot:finished",
							UUID:        uuid.New().String(),
							Body: backups{
								Name:          backupData.Name,
								BucketName:    backupData.BucketName,
								BackupMetrics: backupData.BackupMetrics,
								Snapshots: []snapshot{
									snapshotData,
								},
							},
						}
						addToMessageQueue(singleBackup)
					}
				}
			}
		} else {
			log.Println("unable to handle webhook: %v", backupData)
		}
	}
}

func addToMessageQueue(message webhook) {
	backupMessage, _ := json.Marshal(message)
	err := rabbitChannel.Publish(
		"",
		queueName,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(backupMessage),
		})
	if message.Body.Snapshots != nil {
		log.Printf("webhook for %s, snapshotname %s, ID:%s added to queue", message.Webhooktype+":"+message.Event, message.Body.Snapshots[0].Hostname, message.Body.Snapshots[0].ID)
	} else {
		log.Printf("webhook for %s, ID:%s added to queue", message.Webhooktype+":"+message.Event, message.Body.SnapshotID)
	}
	failOnError(err, "Failed to publish a message")
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Printf("rabbit failure, error is %s:", err.Error())
	}
}

func removeSnapshot(snapshots []snapshot, s int) []snapshot {
	return append(snapshots[:s], snapshots[s+1:]...)
}

func apiBackupInWebhook(slice []snapshot, item string) bool {
	for _, v := range slice {
		if v.ID == item {
			return true
		}
	}
	return false
}
func backupInEnvironment(slice api.Environment, item string) bool {
	for _, v := range slice.Backups {
		if v.BackupID == item {
			return true
		}
	}
	return false
}
