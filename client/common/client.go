package common

import (
	"bufio"
	"fmt"
	"github.com/7574-sistemas-distribuidos/docker-compose-init/client/clientbetinfo"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID               string
	ServerAddress    string
	LoopLapse        time.Duration
	LoopPeriod       time.Duration
	PacketLimit      int
	ServerACK        string
	EndMessageMarker string
}

// Client Entity that encapsulates how
type Client struct {
	config ClientConfig
	conn   net.Conn
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig) *Client {
	client := &Client{
		config: config,
	}
	return client
}

// CreateClientSocket Initializes client socket. In case of
// failure, error is printed in stdout/stderr and exit 1
// is returned
func (c *Client) createClientSocket() error {
	conn, err := net.Dial("tcp", c.config.ServerAddress)
	if err != nil {
		log.Fatalf(
			"action: connect | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
	}
	c.conn = conn
	return nil
}

// StartClientLoop Send messages to the client until some time threshold is met
func (c *Client) StartClientLoop() {
	// autoincremental msgID to identify every message sent
	msgID := 1
	endSignalsChannel := make(chan os.Signal, 1)
	signal.Notify(endSignalsChannel, syscall.SIGTERM)

loop:
	// Send messages if the loopLapse threshold has not been surpassed
	for timeout := time.After(c.config.LoopLapse); ; {
		select {
		case <-timeout:
			log.Infof(
				"action: timeout_detected | result: success | client_id: %v",
				c.config.ID,
			)
			break loop

		case endSignal := <-endSignalsChannel:
			log.Infof("signal '%v' received: shutting down client", endSignal)
			break loop

		default:
		}

		// Create the connection the server in every loop iteration. Send an
		c.createClientSocket()

		// TODO: Modify the send to avoid short-write
		fmt.Fprintf(
			c.conn,
			"[CLIENT %v] Message N°%v\n",
			c.config.ID,
			msgID,
		)
		msg, err := bufio.NewReader(c.conn).ReadString('\n')
		msgID++
		c.conn.Close()

		if err != nil {
			log.Errorf("action: receive_message | result: fail | client_id: %v | error: %v",
				c.config.ID,
				err,
			)
			return
		}
		log.Infof("action: receive_message | result: success | client_id: %v | msg: %v",
			c.config.ID,
			msg,
		)

		// Wait a time between sending one message and the next one
		time.Sleep(c.config.LoopPeriod)
	}

	log.Infof("action: loop_finished | result: success | client_id: %v", c.config.ID)
}

func (c *Client) SendBet(betInfo clientbetinfo.ClientBetInfo) error {
	err := c.createClientSocket()
	if err != nil {
		log.Errorf("cannot send bet, connection FAILED")
		return err
	}

	betAsString := betInfo.ToString()
	betAsString += c.config.EndMessageMarker
	betAsBytes := []byte(betAsString)
	messageLength := len(betAsBytes)

	amountOfMessages := int(messageLength/c.config.PacketLimit) + 1
	log.Debugf("Amount of messages to send: %v", amountOfMessages)
	shortWriteAvoidance := 0

	for messageNum := 0; messageNum < amountOfMessages; messageNum++ {
		log.Debugf("Message %v of %v", messageNum+1, amountOfMessages)

		lowerLimit := messageNum*c.config.PacketLimit - shortWriteAvoidance
		upperLimit := lowerLimit + c.config.PacketLimit

		if upperLimit > messageLength {
			upperLimit = messageLength
		}

		bytesToSend := betAsBytes[lowerLimit:upperLimit]
		bytesSent, err := c.conn.Write(bytesToSend)
		if err != nil {
			return err
		}
		shortWriteAvoidance = len(bytesToSend) - bytesSent
	}

	log.Debugf("Bet from agency %v was sent", betInfo.AgencyID)
	return nil
}

func (c *Client) ListenResponse() error {
	response := make([]byte, 0) // Will contain the response from the server

	for {
		buffer := make([]byte, c.config.PacketLimit)
		bytesRead, err := c.conn.Read(buffer)
		if err != nil {
			log.Errorf("unexpected error while trying to get server response: %w", err)
			return err
		}

		response = append(response, buffer[:bytesRead]...)

		if string(response) == c.config.ServerACK {
			log.Debugf("Got server ACK!")
			break
		}
	}

	return c.conn.Close()
}
