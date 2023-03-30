package betagency

import (
	"github.com/7574-sistemas-distribuidos/docker-compose-init/client/clientbetinfo"
	"github.com/7574-sistemas-distribuidos/docker-compose-init/client/common"
	log "github.com/sirupsen/logrus"
)

type BetAgency struct {
	betInfo clientbetinfo.ClientBetInfo
	client  *common.Client
}

func NewBetAgency(betInfo clientbetinfo.ClientBetInfo, client *common.Client) *BetAgency {
	return &BetAgency{
		betInfo: betInfo,
		client:  client,
	}
}

func (ba *BetAgency) RegisterBet() error {
	log.Debugf("the bet from agency %v will be sent", ba.betInfo.AgencyID)
	err := ba.client.SendBet(ba.betInfo)
	if err != nil {
		log.Errorf("error sending bet: %w", err)
		return err
	}

	log.Debugf("Waiting for server's response to know if the bet was persisted for agency %v", ba.betInfo.AgencyID)
	err = ba.client.ListenResponse()
	if err != nil {
		log.Errorf("error while listening server response: %w", err)
		return err
	}

	log.Infof("action: apuesta_enviada | result: success | dni: %v | numero: %v", ba.betInfo.ClientID, ba.betInfo.Number)
	log.Debug("tudo bem, tudo legal")
	return nil
}
