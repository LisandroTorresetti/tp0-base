import logging
from .utils import *

class Agency:

    def __init__(self, config):
        self.config = config
        self.finishProcessing = False

    '''Persists the bet of a client'''
    def persistBets(self, bets):
        logging.debug(f"bets from client {bets[0].document} is gonna be persisted")
        store_bets(bets)

    '''Returns a Bet object that represents the bet of a client'''
    def getBetFromMessage(self, message):
        agencyID, clientID, name, surname, birthDate, number = message.split(',')
        return Bet(agencyID, name, surname,clientID, birthDate, number)

    '''Process the bet of a client'''
    def RegisterBet(self, client):
        try:
            message = b''
            packetLimit = 8 * 1024 # 8kB

            logging.debug("Listening message from client")

            endMessageMarker = self.config["end_message_marker"].encode()
            allBetsSentMarker = self.config["all_bets_received"].encode()

            while (not endMessageMarker in message) and (not allBetsSentMarker in message):
                actualMessage = client.recv(packetLimit)
                message += actualMessage

            logging.debug("message received!")

            if allBetsSentMarker in message:
                logging.debug("Received FIN message, all bets were processed. Gonna send ACK to client")
                self.SendACK(client)
                self.finishProcessing = True
                return

            message = message.decode('utf-8')
            betsToPersist = []

            for betAsString in message.rstrip("|PING").split(self.config["bets_delimiter"]):
                betsToPersist.append(self.getBetFromMessage(betAsString))

            self.persistBets(betsToPersist)
            self.SendACK(client)

            logging.info(f"action: apuestas_almacenadas | result: success | dni: {betsToPersist[0].document} | amount: {len(betsToPersist)}")

        except OSError:
            logging.debug("OS ERROR")
            raise
        except Exception as e:
            logging.error(f"Other error: {e}")
            raise

    def SendACK(self, client):
        logging.debug("sending ACK to client")
        client.send(self.config["ack"].encode('utf-8'))
