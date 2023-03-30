package common

import (
	"bufio"
	"fmt"
	"github.com/7574-sistemas-distribuidos/docker-compose-init/client/clientbetinfo"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

const endBatchMarker = "PING"

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

func (c *Client) CloseConnection() error {
	return c.conn.Close()
}

func (c *Client) OpenConnection() error {
	return c.createClientSocket()
}

func (c *Client) ListenResponse() (string, error) {
	response := make([]byte, 0) // Will contain the response from the server

	for {
		buffer := make([]byte, c.config.PacketLimit)
		bytesRead, err := c.conn.Read(buffer)
		if err != nil {
			log.Errorf("unexpected error while trying to get server response: %w", err)
			return "", err
		}

		response = append(response, buffer[:bytesRead]...)
		size := len(response)

		if size >= 4 && string(response[size-4:size]) == c.config.ServerACK {
			log.Debugf("Got server ACK!")
			break
		}
	}
	serverResponse := string(response)
	log.Debugf("Response from server: %s", serverResponse)
	return serverResponse, nil
}

func (c *Client) SendBetBatch(betsToSend []clientbetinfo.ClientBetInfo) error {
	// Convert all bets in a string delimited with |, e.g bet1|bet2|bet3|...
	betsAsString := ""
	for _, bet := range betsToSend {
		betStr := bet.ToString()
		betStr += c.config.EndMessageMarker // Here we use the marker as a delimiter of bets
		betsAsString += betStr
	}
	betsAsString += endBatchMarker
	betAsBytes := []byte(betsAsString)
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

	log.Debugf("Bets from agency %v were sent!", betsToSend[0].AgencyID)
	return nil
}

func (c *Client) SendFin() error {
	log.Debug("SENDING FIN MESSAGE")
	finMessage := []byte(c.config.ServerACK) // The server ACK will be our FIN message
	lowerLimit := 0

	for {
		bytesToSend := len(finMessage[lowerLimit:])
		bytesSent, err := c.conn.Write(finMessage[lowerLimit:])
		if err != nil {
			return err
		}
		diff := bytesToSend - bytesSent
		if bytesToSend-bytesSent == 0 {
			log.Debug("Fin message sent")
			break
		}

		lowerLimit = diff
	}

	log.Debug("Waiting for server FIN ACK")
	_, err := c.ListenResponse()
	return err
}

func (c *Client) SendMessage(message string) error {
	log.Debugf("SENDING %s MESSAGE", message)
	finMessage := []byte(message) // The server ACK will be our FIN message
	lowerLimit := 0

	for {
		bytesToSend := len(finMessage[lowerLimit:])
		bytesSent, err := c.conn.Write(finMessage[lowerLimit:])
		if err != nil {
			return err
		}
		diff := bytesToSend - bytesSent
		if bytesToSend-bytesSent == 0 {
			log.Debug("message %s sent", message)
			break
		}

		lowerLimit = diff
	}

	return nil
}

func (c *Client) GetWinnersForAgency(agencyID int) ([]string, error) {
	var winners []string
	for {
		err := c.OpenConnection()
		if err != nil {
			log.Errorf("Agency %v: error opening connection in winners loop: %w")
			return []string{}, err
		}

		getWinnersMessage := fmt.Sprintf("WINNERS|%v|%s", agencyID, c.config.ServerACK) // WINNERS|agencyID|PONG
		err = c.SendMessage(getWinnersMessage)
		if err != nil {
			log.Errorf("Agency %v: error sending get winners message: %w", agencyID, err)
			return []string{}, err
		}

		log.Debugf("Agency %v: waiting for winners", agencyID)
		response, err := c.ListenResponse() // Could be: PROCESSING|PONG or WINNERS|ID1,ID2,ID3|PONG
		if err != nil {
			log.Debugf("Agency %v: error receiving server response about winners: %w", agencyID, err)
			return []string{}, err
		}
		if !keepAsking(response) {
			log.Debugf("Agency %v: got winners %s", agencyID, response)
			log.Debug("Agency %v: parse server response about winners", agencyID)
			winners = parseServerResponse(response)
			break
		}

		log.Debugf("Agency %v: keep asking for winners", agencyID)
		err = c.CloseConnection()
		if err != nil {
			log.Errorf("Agency %v: error clossing connection in winners loop: %w")
			return []string{}, err
		}
		timeToSleep := rand.Intn(100) + 1 // To avoid 0
		time.Sleep(time.Duration(timeToSleep) * time.Microsecond)
	}

	return winners, nil
}

func keepAsking(response string) bool {
	return strings.Split(response, "|")[0] == "PROCESSING"
}

func parseServerResponse(response string) []string {
	winners := strings.Split(response, "|")[1] // Here response is: WINNERS|ID1,ID2,ID3|PONG
	if len(winners) == 0 {
		return []string{}
	}
	return strings.Split(winners, ",")
}
