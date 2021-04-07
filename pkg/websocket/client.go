package websocket

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"time"
)

type Client struct {
	ID       string
	Conn     *websocket.Conn
	Pool     *Pool
	LastPing time.Time
}

type NodeMessage struct {
	Command string `json:"command"`
	NodeId  string `json:"nodeId"`
}

type Message struct {
	Type          int         `json:"type"`
	Content       NodeMessage `json:"content"`
	NodeId        string      `json:"nodeId"`
	RelatedClient *Client     `json:"client"`
}

func (c *Client) Read() {
	defer func() {
		err := c.Conn.Close()
		if err != nil {
			log.Println(err.Error())
		}
		c.Pool.Unregister <- c
	}()

	for {
		messageType, p, err := c.Conn.ReadMessage()
		if err != nil {
			log.Println(err)
			log.Printf("Error occurred while reading message from %s", c.ID)
			break
		}
		//log.Printf("[client.go] Message: %s; Type: %d", string(p), messageType)

		var incomingMessage NodeMessage
		err = json.Unmarshal(p, &incomingMessage)
		if err != nil {
			log.Println(err.Error())
		}

		c.LastPing = time.Now().UTC()
		message := Message{
			Type:          messageType,
			Content:       incomingMessage,
			NodeId:        c.ID,
			RelatedClient: c,
		}

		c.Pool.Send <- message
	}
}
