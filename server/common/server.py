import signal
import socket
import logging
from .agency import *
import threading
from concurrent.futures import ThreadPoolExecutor


class Server:
    def __init__(self, port, listen_backlog, agency):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self.sigtermCalled = False
        self.agency = agency

    def run(self):
        """
        Dummy Server loop

        Server that accept a new connections and establishes a
        communication with a client. After client with communucation
        finishes, servers starts to accept new connections again
        """
        signal.signal(signal.SIGTERM, self.__sigtermHandler)
        writeLock = threading.Lock()
        counterLock = threading.Lock()
        self.agency.InitializeLocks(writeLock, counterLock)

        with ThreadPoolExecutor(max_workers=self.agency.amountOfAgencies + 5) as executor:
            while not self.sigtermCalled:
                try:
                    client_sock = self.__accept_new_connection()
                    executor.submit(self.__handle_client_connection, client_sock)

                except OSError:
                    if self.sigtermCalled:
                        logging.info("signal SIGTERM received, shutting down server")
                    else:
                        raise

    def __sigtermHandler(self, *args):
        self._server_socket.shutdown(socket.SHUT_RDWR)
        self._server_socket.close()
        self.sigtermCalled = True

    def __handle_client_connection(self, client_sock):
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        try:
            finishProcessing = False
            addr = client_sock.getpeername()
            logging.info(f'action: receive_message | result: success | ip: {addr[0]}')

            while not finishProcessing:
                finishProcessing = self.agency.RegisterBet(client_sock)

            logging.info(f'Processed all bets from ip: {addr[0]}')

        except OSError as e:
            logging.error(f"action: receive_message | result: fail | error: {e}")
        finally:
            client_sock.close()

    def __accept_new_connection(self):
        """
        Accept new connections

        Function blocks until a connection to a client is made.
        Then connection created is printed and returned
        """

        # Connection arrived
        logging.info('action: accept_connections | result: in_progress')
        c, addr = self._server_socket.accept()
        logging.info(f'action: accept_connections | result: success | ip: {addr[0]}')
        return c
