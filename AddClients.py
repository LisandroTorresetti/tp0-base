import sys


def getClientsStructure(amountOfClients):
    clients = []
    for clientNum in range(1,amountOfClients + 1):
        client = f"client{clientNum}"

        clientStructure = f"  {client}:\n" \
                          f"    container_name: {client}\n" \
                          f"    image: client:latest\n" \
                          f"    entrypoint: /client\n" \
                          f"    environment:\n" \
                          f"      - CLI_ID={clientNum}\n" \
                          f"      - CLI_LOG_LEVEL=DEBUG\n" \
                          f"    networks:\n" \
                          f"      - testing_net\n" \
                          f"    depends_on:\n" \
                          f"      - server\n"

        clients.append(clientStructure)

    return "\n".join(clients)

def AddClients():
    if len(sys.argv) != 2:
        print("Error: invalid amount of parameters")
        return

    amountOfClients = int(sys.argv[1])
    if amountOfClients <= 0:
        print("Error: the amount of clients to add must be greater than 0")
        return

    with open("template.yaml", 'r') as templateFile:
        with open("docker-compose-dev.yaml", 'w') as resultFile:
            for line in templateFile:
                if "#Clients" in line:
                    clients = getClientsStructure(amountOfClients)
                    resultFile.write(clients)
                    continue
                resultFile.write(line)

if __name__ == "__main__":
    AddClients()





