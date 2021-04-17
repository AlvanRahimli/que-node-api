package main

import (
	"encoding/json"
	"fmt"
	"github.com/AlvanRahimli/que-node-api/pkg/websocket"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

type NodeCommand struct {
	NodeId string `json:"nodeId"`
	Command string `json:"command"`
}

type CustomDateTime struct {
	Title string `json:"title"`
	Date string `json:"date"`
	Time string `json:"time"`
}

func serveWs(pool *websocket.Pool, w http.ResponseWriter, r *http.Request) {
	wsConn, err := websocket.Upgrade(w, r)
	if err != nil {
		log.Error(err.Error())
		_, err2 := fmt.Fprint(w, err.Error())
		if err2 != nil {
			log.Error(err.Error())
		}

		return
	}

	var nodeId string
	for {
		var command NodeCommand
		messageType, p, err := wsConn.ReadMessage()
		log.Infof("[main.go] Message: %s; Type: %d received", string(p), messageType)

		err = json.Unmarshal(p, &command)
		if err != nil {
			log.Errorf("Could not parse first command of node")
			break
		}

		// TODO: Validate & identify node
		nodeId = command.NodeId

		// If command is GetTime - ONLY FIRST TIME
		if command.Command == "getTime" {
			now := time.Now().UTC()
			response := CustomDateTime{
				Date: now.Format("02:01:2006"),
				Time: now.Format("15:04:05"),
				Title: "getTime",
			}

			encoded, err := json.Marshal(response)
			err = wsConn.WriteMessage(1, encoded)
			if err != nil {
				log.Println(err.Error())
				break
			}
		}

		break
	}

	client := &websocket.Client {
		ID: nodeId,
		Conn: wsConn,
		Pool: pool,
		LastPing: time.Now(),
	}

	wsConn.SetPingHandler(func(appData string) error {
		log.Debugf("Node %s pinged server!", client.ID)
		client.LastPing = time.Now().UTC()
		return nil
	})

	pool.Register <- client
	client.Read()
}

func setupRoutes() {
	pool := websocket.NewPool()
	go pool.Start()

	http.HandleFunc("/ws", func(writer http.ResponseWriter, request *http.Request) {
		serveWs(pool, writer, request)
	})
}

func main() {
	fmt.Println("Que Node Api v1")
	log.SetLevel(log.DebugLevel)
	log.SetReportCaller(false)

	log.Info("Application is starting...")

	setupRoutes()

	log.Println("Application started, listening port :8000")
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		log.Error(err.Error())
	}
}
