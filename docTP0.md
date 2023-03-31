# Documentación TP 0

## Protocolo Ping-Pong

El protocolo a lo largo de los ejercicios sufrió ciertas variaciones, por lo que primero detallaré lo que tienen en común cada uno de ellos y luego mencionaré las diferencias:

+ **Bet Encode:** se separan a los atributos de la apuesta con ',' y están ordenados de la siguiente manera: `agencyID,id,name,surname,birthDate,number`.

+ **Bet Delimiter:** el delimitador elegido para separar apuestas concatenadas es el _pipe_ '|', ej: suponiendo que _bet1_ y _bet2_ son dos apuestas encodeadas como se mencionó más arriba, el resultado de la concatenación entre ambas es `bet1|bet2`.

+ **Chunk End Marker:** cuando se manda un _chunk_ de apuestas, para indicar que el chunk finaliza se utiliza el _string_ `PING`. Ej: tenemos un chunk con 3 apuestas _bet1_, _bet2_ y _bet3_, el mensaje que se envia al servidor quedaría como `bet1|bet2|bet|PING`.

+ **All chunks sent:** para indicar que se procesaron todas las apuestas del lado del cliente, se manda un mensaje especial que únicamente dice `PONG`. Podria verse como un proceso similar al cierre de conexión que realiza _TCP_.

+ **Server ACK resonse:** en este caso se usa el mismo string de antes, o sea `PONG`.

+ **Get Winners:** cuando se mandan todas las apuestas, para solicitarle los ganadores al servidor se manda el mensaje `WINNERS|agencyID|PONG`.

+ **Winners Server Response:** para este caso hay dos posibilidades:
	1. Si el servidor sigue procesando apuestas de otras agencias, le manda al cliente el mensaje `PROCESSING|PONG`.
	2. Si ya proceso todas las apuestas, la forma de mandar los ganadores es `WINNERS|id1,id2,...,idn|PONG`. 

A continuación, menciono las variantes en cada ejercicio:

+ **Ejercicio 5:** Como se manda sólo una apuesta en este caso, no se utiliza el `PING`, podría utilizarse pero en el momento de plantear la solución no se lo consideró, sino que la idea surgió en el punto 6. Para este caso, lo que indica que se recibió toda la apuesta es el _bet delimiter_. En este ejercicio tampoco se manda el mensaje final de parte del cliente, el server procesa la apuesta y envia el ACK (`PONG`) que el cliente se queda esperando.

+ **Ejercicio 6:** Como mencioné en el caso anterior, de este ejercicio en adelante se comenzó a utilizar el `PING` para indicar que un chunk finaliza y en este ejercicio comenzó a mandarse el mensaje especial para indicar que se enviaron todas las apuestas, o sea en un paquete se envía sólo un `PONG`.

+ **Ejercicio 7:** En este ejercicio se comienzan a utilizar los mensajes sobre _winners_. Un cambio con respecto al ejercicio anterior es que cuando estoy esperando por la respuesta con los ganadores, las conexiones se abren y se cierran cada cierto tiempo que es aleatorio (de 1ms a 100ms), para que el _server_ pueda seguir procesando apuestas y que no se quede un cliente con la conexión para siempre.

## Concurrencia

Se utilizaron los siguientes elementos para manejar la concurrencia en _Python_:

+ **Locks:** se utilizaron dos locks, uno de escritura para la base de datos, y otro para el contador de la cantidad de agencias que fueron procesadas hasta el momento.

+ **Pool de threads:** la cantidad de workers se definió como `cantidad de agencias + 5` (no hay motivo fijo para sumar 5, simplemente se considera que con esa cantidad vamos a estar más que bien dado que son 5 agencias). Cada thread se encarga de ejecutar el método `__handle_client_connection`.

## Posibles refactors generales

+ Al comienzo se definieron las variables en los archivos de configuracion, pero luego por cuestion de simplicidad para testear se lo agregaron como constantes en los distintos archivos, por lo tanto se podria unificar y que todo tenga variables de entorno.

+ En caso de que ocurra un error del lado del cliente, se lo loggea y se cierra la conexion, en otras palabras no son handleados.

+ Posible _deadletter_: cuando enviamos _chunks_ de apuestas, se podrian almacenar aquellas que tuvieron algun error para luego tratar de insertarlas nuevamente.

+ Dado que la "_data base_" es un archivo de texto, y como ante un error finalizamos la ejecucion, si mandamos 3 chunks de apuestas, de las cuales se registran correctamente todas las de los dos primeros pero falla alguna del tercero, las primeras apuestas fueron persistidas mientras que las otras no. No podemos _rollbackear_ esta situacion. El costo de agregar esta logica seria alto ya que no tenemos todas las ventajas que proporciona una base de datos real.

+ _Scripts_ para validar los resultados de los ganadores, o si suceden otras cosas.

+ _Testing_.

+ En el ejercicio 7, las conexiones se abren y se cierran cuando estamos esperando la respuesta con los ganadores, en el ejercicio 8 se sigue haciendo lo mismo pero podría eliminarse este comportamiento y mantener la conexión abierta hasta que finalice toda la lógica del ejercicio.

## Bugs

A continuación se detallan los bugs que se pueden llegar a encontrar en el código:

+ Cuando enviamos un paquete desde el cliente, puede llegar a pasar que no se envíen todos los _bytes_ si justo ocurre un _short write_ cuando estamos enviando el último. Muestro el código para que se entienda más este caso:

```go
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
```

Suponer que messageNum solo tenemos que mandar un paquete, en ese caso `amountOfMessages = 1`, y que la longitud del paquete es de 10 bytes, pero solo 8 se enviaron. En este caso se da que `shortWriteAvoidance = 2`, ergo se deberían enviar 2 bytes todavía, pero por la condición del _for-loop_ esto no sucede. La solución más sencilla que no implemento para que se mantenga la entrega como lo que se mostró en la presentación es cambiar el loop por uno que no finalice hasta que no se envíen todos los bytes.

+ En ciertas ocasiones, cuando usaba el _makefile_ para levantar la aplicación se obtenían ciertos errores para establecer la conexión _TCP_, creo que no son errores del código dado que se utilizan las mismas funciones que fueron provistas, pero es un bug que ocurre a veces. Haciendo _down_ y luego buildeando de nuevo todo funciona.

+ **Doble mensaje**: no se suele dar, pero pasa que a veces cuando se preguntan por los ganadores se obtiene, por ejemplo, este mensaje del servidor `WINNERS|34963649,35635602|PONGWINNERS|34963649,35635602|PONG` en vez de `WINNERS|34963649,35635602|PONG`. OBS: no sucede siempre, ciertas veces se da.

+ **Minor bugs:** son muchos logs, así que puede pasar que haya un par que tengan mal el formato o algo similar

**OBS:** no se si son todos los bugs, son los que llegué a ver o darme cuenta por _border cases_.

## Run

+ No se utiliza ningun comando especial, solo los del _Makefile_ provisto por la cátedra.

+ Para correr el _script_ que agrega clientes al usuario utilizar `python3 AddClients.py amountOfClients`, con `amountOfClients > 0`, en caso de no cumplir se lanza un error. 

## Observaciones

+ Por accidente borré la _netcat_ en algún commit así que la vuelvo a agregar junto con este documento como un commit que va a figurar en todos los branches que corresponda (la netcat del ejercicio 3 en adelante si corresponde).

+ El script que agrega clientes **NO** setea las variables de entorno del punto 4, tampoco agrega la netcat ni otras variables que puedan llegar a usarse en otros ejercicios.

+ La _netcat_ una vez que pasa el test finaliza su ejecución. Si en cierta cantidad de intentos no se obtiene el _echo_ del mensaje enviado, finaliza loggeando que el test no fue exitoso.

+ Un fix que hice en el ejercicio 8 lo agrego como un commit nuevo en el ejercicio 7. El commit en cuestión es el siguiente: `496de0620b46dd2f8d32ff4d71b24fd3b470a065`