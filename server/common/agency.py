import logging
from .utils import *

class Agency:

    def __init__(self, config):
        self.config = config

    '''Persists the bet of a client'''
    def persistBet(self, bet):
        logging.debug(f"bet from client {bet.document} is gonna be persisted")
        store_bets([bet])

    '''Returns a Bet object that represents the bet of a client'''
    def getBetFromMessage(self, message):
        agencyID, clientID, name, surname, birthDate, number = message.rstrip('|').split(',')
        return Bet(agencyID, name, surname,clientID, birthDate, number)

    '''Process the bet of a client'''
    def RegisterBet(self, client):
        try:
            message = b''
            packetLimit = 8 * 1024 # 8kB

            logging.debug("Listening message from client")
            while not self.config["end_message_marker"].encode() in message:
                actualMessage = client.recv(packetLimit)
                message += actualMessage

            logging.debug("message received")

            message = message.decode('utf-8')
            logging.debug("message content: " + message)
            bet = self.getBetFromMessage(message)

            self.persistBet(bet)

            logging.debug("sending ACK to client")
            client.send(self.config["ack"].encode('utf-8'))

            logging.info(f"action: apuesta_almacenada | result: success | dni: {bet.document} | numero: {bet.number}")
        except OSError:
            logging.debug("OS ERROR")
        except Exception as e:
            logging.error(f"Other error: {e}")
