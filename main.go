package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"gopkg.in/olivere/elastic.v6"
)

type OnlineModels []OnlineModel

var puddinPublic = false

// CB affiliate identifier
var affId string
var alertRoom string
var notificationChannelId string
var viewerNotificationChannelId string
var discordBotToken string
var esClient *elastic.Client

func main() {
	affId = os.Getenv("PUDDIN_AFF_ID")
	alertRoom = os.Getenv("PUDDIN_ALERT_ROOM")
	notificationChannelId = os.Getenv("PUDDIN_NOTIFICATION_CHANNEL_ID")
	viewerNotificationChannelId = os.Getenv("PUDDIN_VIEWER_NOTIFICATION_CHANNEL_ID")
	discordBotToken = os.Getenv("PUDDIN_DISCORD_BOT_TOKEN")

	ctx, cancel := context.WithCancel(context.Background())

	esClient = getElasticClient()

	createOnlineRoomIndex(esClient)
	createViewerIndex(esClient)
	createSessionIndex(esClient)

	startDiscord()
	defer closeDiscord()

	go watchOnlineRooms(affId, esClient, ctx)
	go logViewers(affId, esClient, ctx)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	<-sigChan
	cancel()
}
