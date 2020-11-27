# Step-by-step guide

This manual will guide you through the async dkg ceremony. That is a part of Lido DAO testnet launch. Note that async ceremony goal is to demo the ceremony before the mainnet launch, so the withdrawal credentials generated during this test run wouldn't be used in test or any other deployments.

## Preparations 

You need to set up two machines — internet-connected client and an airgapped signer. Please, follow [this guide](https://github.com/lidofinance/dc4bc/blob/master/TESTNET_CLIENT_PREP_GUIDE.md) to build the code and prepare it for the ceremony: build the apps, generate keypairs and send pubkeys as a pull to the repo.

Distributed key generation nodes use Kafka to communicate. The testnet dev team will provide you the Kafka address upon the ceremony start. Along with the address, we'll send you a Kafka topic for the ceremony and a certificate to communicate with the node.

## DKG ceremony

Exact commands are detailed in the [dc4bc howto](https://github.com/lidofinance/dc4bc/blob/master/HowTo.md). The high-level ceremony outline is:
1. Preparations
   1. Set up the apps and generate the keypairs
   2. Save your pubkeys somewhere for the future use
   3. Send the pubkeys as a pull to the repo https://github.com/lidofinance/dc4bc-conference-call/blob/master/dc4bc-conference-call/dc4bc-async-ceremony-27-11-2020.json
2. Gathering the data
   1. Wait for all participants to submit their keys and get the the dkg keys [proposal `json`](https://github.com/lidofinance/dc4bc-conference-call/blob/master/dc4bc-conference-call/dc4bc-async-ceremony-27-11-2020.json)
   2. Get the Kafka node address, topic and certificate from the testnet team
3. Ceremony: generating the signature
   1. Open the apps: dkg node to communicate requests (look up `dc4bc_d start` command), dkg client to format transactions (`dc4bc_cli` app comands) and dkg airgapped signer (`dc4bc_airgapped` console) to sign transactions
   2. Prepare the starting transaction on the client (`dc4bc_cli start_dkg` command) and get the gif with qr codes — it's the transaction data to sign on the airgapped machine. To communicate between the client and the airgapped apps you'll need to record a video with qr gif on your phone from one machine and show it to the other one
   3. Sign the transaction on the airgapped machine (run `read_qr` in the `dc4bc_airgapped` console), get back another qr gif. 
   4. Pass the signed qr gif to the client machine (`dc4bc_cli read_qr`). This will send the signed transaction from the client to kafka node
   5. Sign and send back all the operations between client and airgapped machines
   6.  Get the `event_dkg_master_key_confirm_received` event signaling the signature generation
4.  Ceremony: checking the signature
    1.  Choose the user to sign a message
    2.  The user signing a message uses [this guide](https://github.com/lidofinance/dc4bc/blob/master/HowTo.md#signature) to send the signed message to other ceremony participants
    3.  To get the signature id run `show_finished_dkg` in the airgapped console
    4.  All users check their operation logs (`dc4bc_cli get_operations`) and sign messages until the `signature_reconstructed` event is received
    5.  Users get their signature id (`show_finished_dkg` in `dc4bc_airgapped` console)
    6.  Get the actual signature with client app: `dc4bc_cli get_signatures <FINISHED_DKG_ID>`
    7.  Verify the signature by running `verify_signature` command inside the `dc4bc_airgapped` console
5.  Ceremony ends, the validated signature can be used for the DAO withdrawal credentials
