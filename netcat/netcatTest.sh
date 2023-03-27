#!/bin/bash

testMessage="this is a test message"
serverPort=12345
maxAttempts=50

attempt=1
while [ $attempt -le $maxAttempts ]
do
  echo "Attempt ${attempt}"
  result=$(echo "$testMessage" | nc server $serverPort)
  if [ "$result" = "$testMessage" ]; then
    echo "Netcast test PASSED"
    exit 0
  fi
  sleep 15
  attempt=$(( attempt + 1 ))
done
echo "Netcast test FAILED"

