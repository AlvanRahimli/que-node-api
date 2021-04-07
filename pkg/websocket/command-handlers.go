package websocket

import (
	"database/sql"
	"encoding/json"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"time"
)

func (c *Client) handleExpiredOtp(dbContext *sql.DB, otp string) error {
	stmt, err := dbContext.Prepare("update QueueElements set State = 2 where Otp = ? and NodeId = ?")
	if err != nil {
		log.Println(err.Error())
		return err
	}

	_, err = stmt.Query(otp, c.ID)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	log.Printf("EXPIRED OTP ENDPOINT QUERY with Otp: %s", otp)

	err = c.handleGetNearest(dbContext)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	return nil
}

func (c *Client) handleValidOtp(dbContext *sql.DB, otp string) error {
	stmt, err := dbContext.Prepare("update QueueElements set Expired = true, State = 1 where Otp = ? and NodeId = ?")
	if err != nil {
		log.Println(err.Error())
		return err
	}

	_, err = stmt.Query(otp, c.ID)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	log.Println("HANDLE VALID OTP QUERY")

	err = c.handleGetNearest(dbContext)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	return nil
}

func (c *Client) handleLastPerson(number int) error {
	log.Printf("Node %s added %d numbered man to queue", c.ID, number)
	return nil
}

func (c *Client) handleGetNearest(dbContext *sql.DB) error {
	stmt, err := dbContext.Prepare("SELECT QueueElements.Timestamp, QueueElements.Otp FROM QueueElements WHERE NodeId=? AND Type != 1 AND State = 0 ORDER BY QueueElements.Timestamp LIMIT 1")
	if err != nil {
		log.Println(err.Error())
		return err
	}

	dbResult, err := stmt.Query(c.ID)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	log.Infof("HANDLE GET NEAREST QUERY for Node %s", c.ID)

	var response NextPerson
	for dbResult.Next() {
		err = dbResult.Scan(&response.Time, &response.Otp)
		if err != nil {
			log.Println(err.Error())
			break
		}

		parsedTime, err := time.Parse("2006-01-02 15:04:05", response.Time)

		if err != nil {
			log.Println(err.Error())
			break
		}
		response.Time = parsedTime.Format("15:04:05")
	}

	response.Title = "getNearest"

	writer, writerCreateError := c.Conn.NextWriter(1)
	if writerCreateError != nil {
		log.Fatalln(writerCreateError)
	}

	encodeError := json.NewEncoder(writer).Encode(response)
	if encodeError != nil {
		log.Error(encodeError)
	}

	return nil
}

func handleGetTime(conn *websocket.Conn) error {
	now := time.Now().UTC()
	response := CustomDateTime{
		Date:  now.Format("02:01:2006"),
		Time:  now.Format("15:04:05"),
		Title: "getTime",
	}

	writer, writerCreateError := conn.NextWriter(1)
	if writerCreateError != nil {
		log.Fatalln(writerCreateError)
	}

	encodeError := json.NewEncoder(writer).Encode(response)
	if encodeError != nil {
		log.Error(encodeError)
	}

	return nil
}
