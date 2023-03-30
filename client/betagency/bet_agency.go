package betagency

import (
	"bufio"
	"fmt"
	"github.com/7574-sistemas-distribuidos/docker-compose-init/client/clientbetinfo"
	"github.com/7574-sistemas-distribuidos/docker-compose-init/client/common"
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
)

const (
	fileFormat = ".csv"
)

type BetAgency struct {
	ID        int
	betInfo   clientbetinfo.ClientBetInfo
	chunkSize int
	client    *common.Client
}

func NewBetAgency(agencyID int, client *common.Client, chunkSize int) *BetAgency {
	return &BetAgency{
		ID:        agencyID,
		chunkSize: chunkSize,
		client:    client,
	}
}

func (ba *BetAgency) ProcessBatch() error {
	err := ba.client.OpenConnection()
	if err != nil {
		log.Errorf("cannot send bet, connection FAILED")
		return err
	}
	defer func() {
		err = ba.client.CloseConnection()
		if err != nil {
			log.Errorf("error closing client connection from agency %v: %w", ba.ID, err)
		}
	}()

	betsFilepath := getFilePath(ba.ID)
	betsFile, err := os.Open(betsFilepath)
	if err != nil {
		log.Debugf("error opening bets file from agency %v: %w", ba.ID, err)
	}

	defer betsFile.Close()

	fileScanner := bufio.NewScanner(betsFile)
	fileScanner.Split(bufio.ScanLines)

	betsCounter := 0
	batchesSent := 0
	var betsToSend []string
	for fileScanner.Scan() {
		if betsCounter == ba.chunkSize {
			batchesSent += 1
			log.Debugf("Sending batch number %v for agency %v", batchesSent, ba.ID)
			err = ba.RegisterBets(betsToSend)
			if err != nil {
				log.Errorf("error sending batch for agency %v: %w", ba.ID, err)
				return err
			}
			betsCounter = 0
			betsToSend = []string{}
		}
		line := fileScanner.Text()
		if len(line) < 2 {
			// sanity check
			break
		}
		betsToSend = append(betsToSend, line)
		betsCounter += 1
	}
	if len(betsToSend) != 0 {
		log.Debugf("Sending batch number %v for agency %v", batchesSent+1, ba.ID)
		err = ba.RegisterBets(betsToSend)
		if err != nil {
			log.Errorf("error sending batch for agency %v: %w", ba.ID, err)
			return err
		}
	}

	err = ba.client.SendFin()
	if err != nil {
		log.Errorf("error sending FIN message from agency %v: %w", ba.ID, err)
		return err
	}
	log.Debugf("All bets from agency %v were sent. Process batch status: FINISHED", ba.ID)
	err = ba.client.CloseConnection()
	if err != nil {
		log.Debugf("Agecy %v: error closing client's connection: %w", err)
		return err
	}

	log.Debugf("Agency %v: waiting for winners", ba.ID)
	winners, err := ba.client.GetWinnersForAgency(ba.ID)
	if err != nil {
		log.Errorf("Agency %v: error getting winners: %w", ba.ID, err)
		return err
	}

	log.Debugf("Agency %v: winner IDs: %v", ba.ID, winners)
	log.Infof("action: consulta_ganadores | agencia: %v | result: success | cant_ganadores: %v", ba.ID, len(winners))
	return nil
}

func (ba *BetAgency) RegisterBets(betsToSend []string) error {
	// Convert to ClientBetInfo each bet
	var bets []clientbetinfo.ClientBetInfo
	for _, b := range betsToSend {
		bet := clientbetinfo.FromStringToBet(ba.ID, b)
		bets = append(bets, bet)
	}

	// Send bets
	log.Debugf("Agency %v: Sending bets...", ba.ID)
	err := ba.client.SendBetBatch(bets)
	if err != nil {
		return err
	}

	// Wait for server response
	log.Debugf("Agency %v: Waiting for server's ack", ba.ID)
	_, err = ba.client.ListenResponse()
	if err != nil {
		log.Errorf("Agency %v: error while waiting for server's ACK: %w", ba.ID, err)
	}

	return nil
}

func (ba *BetAgency) RegisterBet() error {
	log.Debugf("the bet from agency %v will be sent", ba.betInfo.AgencyID)
	err := ba.client.SendBet(ba.betInfo)
	if err != nil {
		log.Errorf("error sending bet: %w", err)
		return err
	}

	defer func(client *common.Client) {
		err := client.CloseConnection()
		if err != nil {
			log.Error("error closing client connection")
		}
	}(ba.client)

	log.Debugf("Waiting for server's response to know if the bet was persisted for agency %v", ba.betInfo.AgencyID)
	_, err = ba.client.ListenResponse()
	if err != nil {
		log.Errorf("error while listening server response: %w", err)
		return err
	}

	log.Infof("action: apuesta_enviada | result: success | dni: %v | numero: %v", ba.betInfo.ClientID, ba.betInfo.Number)
	log.Debug("tudo bem, tudo legal")
	return nil
}

// getFilePath returns the path to the .csv file for the bet agency
func getFilePath(agencyID int) string {
	agencyName := fmt.Sprintf("agency-%v%s", agencyID, fileFormat)
	return filepath.Join("/dataset", agencyName)
}
