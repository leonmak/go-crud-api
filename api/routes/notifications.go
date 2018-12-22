package routes

import (
	"context"
	"firebase.google.com/go/messaging"
	"fmt"
	"groupbuying.online/api/env"
	"groupbuying.online/api/utils"
	"log"
	"net/http"
)

func pushNewChatNotification(w http.ResponseWriter, r *http.Request) {
	// params: senderUserId, receiverFirId, senderDisplayName, senderFirId, messageText
	result, err := utils.ReadRequestToJson(r)
	if err != nil {
		log.Fatalf("error getting json body %s", err)
	}
	// validate user is sender
	senderUserId := result["senderUserId"].(string)
	userId, ok := utils.GetUserIdInSession(r)
	if !ok || senderUserId != userId {
		log.Fatalf("error auth")
	}

	ctx := context.Background()
	client, err := env.Firebase.Messaging(ctx)
	if err != nil {
		log.Fatalf("error getting Messaging client %s", err)
	}
	senderFirId := result["senderFirId"].(string)
	receiverFirId := result["receiverFirId"].(string)
	senderDisplayName := result["senderDisplayName"].(string)
	messageText := result["messageText"].(string)

	if err != nil {
		log.Fatalln("Error initializing database client:", err)
	}
	message := &messaging.Message{
		Data: map[string]string{
			"senderFirId": senderFirId,
			"kind": "ChatNotification",
		},
		Notification: &messaging.Notification{
			Title: fmt.Sprintf("%s sent a new message", senderDisplayName),
			Body: messageText,
		},
		APNS: &messaging.APNSConfig{
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					ContentAvailable: true,  // for app to increase badge count in background mode in AppDelegate
				},
			},
		},
		Topic: receiverFirId,
	}
	response, err := client.Send(ctx, message)
	if err != nil {
		log.Fatalln(err)
	}
	utils.WriteSuccessJsonResponse(w, fmt.Sprint("Successfully sent message:", response))
}
