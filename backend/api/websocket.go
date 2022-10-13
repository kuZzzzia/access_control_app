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
		case peopleNumber := <-ctrl.peopleNumberNotification:
			log.Debug().Msg("needNotification")
			ctrl.writeMessage(peopleNumber)
			// if err != nil {
			// 	log.Error().Err(err).Msg("write close:", err)
			// 	return
			// }
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

func (ctrl *Controller) PushPeopleNumber(ctx context.Context, peopleNumber int) {
	client, err := ctrl.App.Messaging(ctx)

	anwswer, err := client.Send(ctx, &messaging.Message{
		Topic: "Notification",
		Notification: &messaging.Notification{
			Title: "People Number",
			Body:  strconv.Itoa(peopleNumber),
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("Marshal")
		return
	}
	log.Debug().Str("anwswer", anwswer).Msg("client.Send")

	// reqUrl := *ctrl.FireBaseUrl
	// reqUrl, err := url.Parse("https://fcm.googleapis.com/v1/projects/access-control-app-986f4/messages:send")

	// body, err := json.Marshal(FireBaseRequest{
	// 	Message: Message{
	// 		Notification: Notification{
	// 			Title: "People Number",
	// 			Body:  strconv.Itoa(peopleNumber),
	// 		},
	// 	},
	// })
	// if err != nil {
	// 	log.Error().Err(err).Msg("Marshal")
	// 	return
	// }

	// req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqUrl.String(), bytes.NewReader(body))
	// if err != nil {
	// 	log.Error().Err(err).Msg("NewRequestWithContext")
	// 	return
	// }

	// req.Header.Set("Authorization", "Bearer AIzaSyD8jIew6NOCmiLe7wCFeBpe-KgY3aQ2zAM")
	// req.Header.Set("Content-Type", "application/json")

	// resp, err := ctrl.HTTPClient.Do(req)
	// if err != nil {
	// 	log.Error().Err(err).Msg("do")
	// 	return
	// }
	// defer resp.Body.Close()

	// if resp.StatusCode != http.StatusNoContent {
	// 	log.Error().Err(errors.New(resp.Status)).Msg("status code")
	// 	return
	// }
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
	// {
	// 	"message":{
	// 	   "token":"token_1",
	// 	   "data":{},
	// 	   "notification":{
	// 		 "title":"People Number",
	// 		 "body":"6",
	// 	   }
	// 	}
	//  }
}
