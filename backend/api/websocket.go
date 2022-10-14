package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"firebase.google.com/go/messaging"
	"github.com/gorilla/websocket"
	"github.com/kuZzzzia/access_control_app/backend/specs"
	"github.com/rs/zerolog/log"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Пропускаем любой запрос
	},
}

func (ctrl *Controller) Ping(w http.ResponseWriter, r *http.Request) {
	connection, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("Upgrade")
	}
	defer connection.Close()

	ctrl.clients[connection] = true
	defer delete(ctrl.clients, connection)

	for {
		mt, message, err := connection.ReadMessage()
		if err != nil || mt == websocket.CloseMessage {
			log.Error().Err(err).Msg("ReadMessage")
			break
		}

		go ctrl.writeMessage(0)

		go messageHandler(message)
	}

}

func (ctrl *Controller) GetPeopleNumber(w http.ResponseWriter, r *http.Request) {
	connection, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("Upgrade")
	}
	defer connection.Close()

	ctrl.clients[connection] = true
	defer delete(ctrl.clients, connection)

	for {
		mt, message, err := connection.ReadMessage()
		if err != nil || mt == websocket.CloseMessage {
			log.Error().Err(err).Msg("ReadMessage")
			break
		}
		messageHandler(message)
		select {
		// case peopleNumber := <-ctrl.peopleNumberNotification:
		// 	log.Debug().Msg("needNotification")
		// 	ctrl.writeMessage(peopleNumber)
		}
	}
}

func (ctrl *Controller) writeMessage(peopleNumber int) {
	for conn := range ctrl.clients {
		message, err := json.Marshal(specs.GetPeopleNumberResponse{
			PeopleNumber: peopleNumber,
		})
		if err != nil {
			log.Error().Err(err).Msg("write answer")
		}

		conn.WriteMessage(websocket.TextMessage, message)
	}
}

func messageHandler(message []byte) {
	fmt.Println(string(message))
}

func (ctrl *Controller) Auth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	firebaseToken := r.Header.Get("firebaseToken")

	err := ctrl.srv.AddNotificationToken(ctx, firebaseToken)
	if err != nil {

		log.Error().Err(err).Msg("failed to add notification token")
	}

	w.WriteHeader(http.StatusOK)
}

func (ctrl *Controller) PushPeopleNumber(ctx context.Context, peopleNumber int) {
	tokens, err := ctrl.srv.ListNotificationTokens(ctx)
	if err != nil {
		log.Error().Err(err).Msg("ListNotificationTokens")
		return
	}

	client, err := ctrl.App.Messaging(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Messaging")
		return
	}

	for i := range tokens {
		anwswer, err := client.Send(ctx, &messaging.Message{
			Token: tokens[i],
			Topic: "Notification",
			Notification: &messaging.Notification{
				Title: "People Number",
				Body:  strconv.Itoa(peopleNumber),
			},
		})
		if err != nil {
			log.Error().Err(err).Msg("Send")
			continue
		}
		log.Debug().Str("anwswer", anwswer).Msg("client.Send")
	}
}

type Notification struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type Message struct {
	Token        string
	Notification Notification
}

type FireBaseRequest struct {
	Message `json:"message"`
}
