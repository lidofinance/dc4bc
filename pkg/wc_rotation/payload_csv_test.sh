#!/bin/bash

WITHDRAWAL_CREDENTIALS="0x009690e5d4472c7c0dbdf490425d89862535d2a52fb686333f3a0a9ff5d2125e"
echo "Please provide provide consensus layer node host."
read -r NODE_HOST

if [[ $NODE_HOST == '' ]]
then
 echo "It's impossible to make a request without ETH node host. Please, provide it";
 exit;
fi

URL="$NODE_HOST/eth/v1/beacon/states/head/validators"
echo "Request url is $URL"

RESPONSE=($(curl --request GET --url "$URL" --header 'Content-Type: application/json'))

echo "Validating payloads.csv"

JQ_FILTER=".data[] | select(.validator.withdrawal_credentials == \"$WITHDRAWAL_CREDENTIALS\") | .index | tonumber"
SORTED_VALIDATORS=$(jq -r "$JQ_FILTER" <<< $RESPONSE | jq -s | jq '.|sort')

jq -r '.[]' <<< "$SORTED_VALIDATORS" > actual.csv

## Checks

jq '.|{ first_validator_index: min, last_validator_index: max, total_validators: length}' <<< "$SORTED_VALIDATORS"
DIFF=$(diff payloads.csv actual.csv)
if [ "$DIFF" != "" ]
then
  echo "actual.csv is not the same payloads.csv $DIFF"
else
  echo "actual.csv is the same payloads.csv"
fi
