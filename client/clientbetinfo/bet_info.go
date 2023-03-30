package clientbetinfo

import (
	"fmt"
	"os"
	"strconv"
)

const (
	agencyIDEnv  = "CLI_ID"
	clientIDEnv  = "DOCUMENTO"
	nameEnv      = "NOMBRE"
	surnameEnv   = "APELLIDO"
	birthDateEnv = "NACIMIENTO"
	numberEnv    = "NUMERO"
)

type ClientBetInfo struct {
	AgencyID  int
	ClientID  int
	Name      string
	Surname   string
	BirthDate string
	Number    int
}

func GetBetInfo() (ClientBetInfo, error) {

	agencyID, err := strconv.Atoi(os.Getenv(agencyIDEnv))
	if err != nil {
		return ClientBetInfo{}, fmt.Errorf("agency id must be numeric")
	}

	clientID, err := strconv.Atoi(os.Getenv(clientIDEnv))
	if err != nil {
		return ClientBetInfo{}, fmt.Errorf("client id must be numeric")
	}

	number, err := strconv.Atoi(os.Getenv(numberEnv))
	if err != nil {
		return ClientBetInfo{}, fmt.Errorf("number must be numeric")
	}

	return ClientBetInfo{
		AgencyID:  agencyID,
		ClientID:  clientID,
		Name:      os.Getenv(nameEnv),
		Surname:   os.Getenv(surnameEnv),
		BirthDate: os.Getenv(birthDateEnv),
		Number:    number,
	}, nil
}

func (cbi ClientBetInfo) ToString() string {
	return fmt.Sprintf(
		"%v,%v,%s,%s,%s,%v",
		cbi.AgencyID,
		cbi.ClientID,
		cbi.Name,
		cbi.Surname,
		cbi.BirthDate,
		cbi.Number,
	)
}
