import logging
import threading

from .utils import *

PACKET_LIMIT = 8 * 1024 # 8kB
class Agency:

    def __init__(self, config):
        self.config = config
        self.betsDelimiter = self.config["bets_delimiter"]
        self.allBetsReceivedMarker = config["all_bets_received"]
        self.endMessageMarker = config["end_message_marker"]
        self.amountOfAgencies = config["amount_of_agencies"]
        self.processAction = config["process_action"] # ToDo: delete this variable we dont use it
        self.winnersAction = config["winners_action"]
        self.agenciesProcessed = 0
        self.finishProcessing = False

        self.writeLock = None # For the fake DB
        self.counterLock = None # For the counter of processed agencies

    '''Persists the bet of a client'''
    def persistBets(self, bets):
        logging.debug(f"bets from client {bets[0].document} is gonna be persisted")

        logging.debug("Acquiring write lock")
        self.writeLock.acquire()

        store_bets(bets)

        logging.debug("Releasing write lock")
        self.writeLock.release()

    '''Returns a Bet object that represents the bet of a client'''
    def getBetFromMessage(self, message):
        agencyID, clientID, name, surname, birthDate, number = message.split(',')
        return Bet(agencyID, name, surname,clientID, birthDate, number)

    '''Process the bet of a client'''
    def RegisterBet(self, client):
        try:
            message = b''
            logging.debug("Listening message from client")

            while (not self.endMessageMarker.encode() in message) and (not self.allBetsReceivedMarker.encode() in message):
                actualMessage = client.recv(PACKET_LIMIT)
                message += actualMessage

            logging.debug("message received!")
            message = message.decode('utf-8')

            if self.allBetsReceivedMarker in message:
                logging.debug(f"Message with action: {message}")
                action = message.split("|")[0] # WINNERS or PONG

                if action == self.winnersAction:
                    logging.debug("A client is asking for the winners")
                    self.SendWinners(client, message)
                    return True

                if action == self.config["ack"]:
                    logging.debug("Received FIN message, all bets were processed. Gonna send ACK to client")
                    self.SendResponse(client, self.config["ack"])
                    self.UpdateCounter()
                    return True

                logging.debug("invalid action got: " + action)
                raise Exception

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

    def SendResponse(self, client, response):
        logging.debug(f"sending {response} to client")
        responseEncoded = response.encode('utf-8')
        client.send(responseEncoded)
        for lowerLimit in range(0, len(responseEncoded), PACKET_LIMIT):
            bytesSent = client.send(responseEncoded[lowerLimit:lowerLimit + PACKET_LIMIT])
            lowerLimit -= PACKET_LIMIT - bytesSent

    def SendWinners(self, client, message):
        logging.debug("Acquiring lock to know if we have to send winners")
        self.counterLock.acquire()
        if self.agenciesProcessed < self.amountOfAgencies:
            logging.debug("Releasing lock to know if we have to send winners")
            self.counterLock.release()
            logging.debug("Still processing agencies...")
            response = "PROCESSING" + "|" + self.config["ack"]
            client.send(response.encode('utf-8'))
            return

        logging.debug("Releasing lock to know if we have to send winners")
        self.counterLock.release()

        logging.info("action: sorteo | result: success")
        agencyID = message.split("|")[1] # Message is WINNERS|agencyID|PONG
        logging.debug(f"Getting winners for agency {agencyID}")
        winners = self.getWinnersForAgencyID(agencyID)
        winnersConcat = ",".join(winners)
        response = self.winnersAction + "|" + winnersConcat + "|" + self.config["ack"] # WINNERS|id1,id2,id3|PONG
        self.SendResponse(client, response)

    '''Returns the winners of the agency with ID agencyID'''
    def getWinnersForAgencyID(self, agencyID):
        winners = []
        self.writeLock.acquire()
        bets = load_bets()
        self.writeLock.release()
        for bet in bets:
            if bet.agency == int(agencyID) and has_won(bet):
                logging.debug(f"winner with doc {bet.document}")
                winners.append(bet.document)

        return winners

    def InitializeLocks(self, writeLock, counterLock):
        self.writeLock = writeLock
        self.counterLock = counterLock

    def UpdateCounter(self):
        logging.debug("Acquiring counter lock")
        self.counterLock.acquire()

        self.agenciesProcessed += 1
        self.agenciesProcessed = min(self.agenciesProcessed, self.amountOfAgencies) # Sanity check

        logging.debug("Releasing counter lock")
        self.counterLock.release()

