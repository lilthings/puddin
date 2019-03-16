package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

type OnlineModels []OnlineModel

var puddinPublic = false

// CB affiliate identifier
var affId string
var alertRoom string
var notificationChannelId string
var viewerName string
var viewerNotificationChannelId string
var discordBotToken string

func main() {
	affId = os.Getenv("PUDDIN_AFF_ID")
	alertRoom = os.Getenv("PUDDIN_ALERT_ROOM")
	notificationChannelId = os.Getenv("PUDDIN_NOTIFICATION_CHANNEL_ID")
	viewerName = os.Getenv("PUDDIN_VIEWER_NAME")
	viewerNotificationChannelId = os.Getenv("PUDDIN_VIEWER_NOTIFICATION_CHANNEL_ID")
	discordBotToken = os.Getenv("PUDDIN_DISCORD_BOT_TOKEN")

	ctx, cancel := context.WithCancel(context.Background())

	client := getElasticClient()

	createOnlineRoomIndex(client)
	createViewerIndex(client)

	startDiscord()
	defer closeDiscord()

	go watchOnlineRooms(affId, client, ctx)
	go logViewers(affId, client, ctx)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	<-sigChan
	cancel()
}
