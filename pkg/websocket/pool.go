package websocket

import (
	"database/sql"
	"encoding/json"
	_ "github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
	"strconv"
	"strings"
	"time"
)

type Pool struct {
	Register   chan *Client
	Unregister chan *Client
	Clients    map[*Client]bool
	Send       chan Message
}

func NewPool() *Pool {
	return &Pool{
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Clients:    make(map[*Client]bool),
		Send:       make(chan Message),
	}
}

type CustomDateTime struct {
	Title string `json:"title"`
	Date  string `json:"date"`
	Time  string `json:"time"`
}

type NextPerson struct {
	Title string `json:"title"`
	Time string `json:"time"`
	Otp string `json:"otp"`
}

func (pool *Pool) Start() {
	log.Info("Pool initialized successfully")
	pool.configurePoolCleanup()

	log.Debug("Using singleton Database Context. Research this topic")
	dbContext, err := sql.Open("mysql", "alvan:01122001_Alvan@tcp(104.155.175.68:3306)/QueDb")
	if err != nil {
		log.Error(err.Error())
	}
	log.Info("Db context created")

	for {
		select {
		case client := <-pool.Register:
			pool.Clients[client] = true
			log.Infof("New Node Connected with ID: %s", client.ID)
			log.Infof("Size of Connection Pool: %d", len(pool.Clients))
			// TODO: Send message to activate node
			break
		case client := <-pool.Unregister:
			delete(pool.Clients, client)
			log.Infof("Node Disconnected with ID: %s", client.ID)
			log.Infof("Size of Connection Pool: %d", len(pool.Clients))
			// TODO: Send message to deactivate node
			break
		case message := <-pool.Send:
			messageJson, err := json.Marshal(message.Content)
			if err != nil {
				log.Errorf("Could not parse message for NodeId: %s\n\t%s", message.NodeId, err.Error())
			}
			log.Debugf("Pool received message %s", messageJson)

			// Handling commands:
			switch message.Content.Command {
			case "getTime":
				err := handleGetTime(message.RelatedClient.Conn)
				if err != nil {
					log.Error("Could not handle getTime request for NodeId: %s\n\t%s", message.NodeId, err.Error())
					break
				}
				break
			case "getNearest":
				err := message.RelatedClient.handleGetNearest(dbContext)
				if err != nil {
					log.Error("Could not handle getNearest request for NodeId: %s\n\t%s", message.NodeId, err.Error())
					break
				}
			default:
				parsed := strings.Split(message.Content.Command, ":")
				switch parsed[0] {
				case "lastPerson":
					number, err := strconv.Atoi(parsed[1])
					if err != nil {
						log.Errorf("Could not parse given number for NodeId: %s \n\t%s", message.NodeId, err.Error())
					}
					err = message.RelatedClient.handleLastPerson(number)
					if err != nil {
						log.Errorf("Could not handle lastPerson request for NodeId: %s\n\t%s", message.NodeId, err.Error())
					}
					break
				case "validOtp":
					err := message.RelatedClient.handleValidOtp(dbContext, parsed[1])
					if err != nil {
						log.Errorf("Could not handle validOtp request for NodeId: %s\n\t%s", message.NodeId, err.Error())
					}
					break
				case "expiredOtp":
					err := message.RelatedClient.handleExpiredOtp(dbContext, parsed[1])
					if err != nil {
						log.Errorf("Could not handle expiredOtp request for NodeId %s\n\t%s", message.NodeId, err.Error())
					}
					break
				}
				break
			}

			message.RelatedClient.LastPing = time.Now().UTC()
		}
	}
}

func (pool *Pool) configurePoolCleanup() {
	go (func() {
		for {
			for client := range pool.Clients {
				nowTime := time.Now().UTC()
				lastPingIsAhead := nowTime.Sub(client.LastPing).Seconds()
				if lastPingIsAhead >= 60 {
					log.Printf("Client %s is behind by %f seconds", client.ID, lastPingIsAhead)
					//err := client.Conn.Close()
					//if err != nil {
					//	log.Printf("Could not close connection with dead node (%s)\n\t%s", client.ID, err.Error())
					//}
					pool.Unregister <- client
				}
			}
			time.Sleep(10 * time.Second)
		}
	})()
	log.Info("Pool cleanup configured")
}
