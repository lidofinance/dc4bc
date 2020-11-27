# Step-by-step guide

Clone the project repository:
```
git clone git@github.com:lidofinance/dc4bc.git
```

#### Installation (Linux)

First install the Go toolchain:
```
curl -OL https://golang.org/dl/go1.15.2.linux-amd64.tar.gz
tar -C /usr/local -xzf go1.15.2.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```

Install Java to generate certificate's truststore for Kafka:
```
sudo apt install default-jre
```

Then build the project binaries:
```
# Go to the cloned repository.
cd dc4bc
make build-linux
```

#### Installation (Darwin)

First install the Go toolchain:
```
mkdir $HOME/Go
export GOPATH=$HOME/Go
export GOROOT=/usr/local/opt/go/libexec
export PATH=$PATH:$GOPATH/bin
export PATH=$PATH:$GOROOT/bin
brew install go
```

Then build the project binaries:
```
# Go to the cloned repository.
cd dc4bc
make build-darwin
```

#### Starting Kafka (optional)

##### Requirements

* Docker

At first, you need to change config files for your Kafka node and TLS certificate generation:

* kafka-docker/.env
* kafka-docker/ca.cnf - important fields here are 'commonName', 'IP.1', 'DNS.1'. IP.1 is your server IP and 'commonName/'DNS.1' is your domain name if you have one.

After that just run in kafka-docker folder:
```
$ ./up.sh
```
If everything is all right, output will be like:
```
Generating a 2048 bit RSA private key
................................................................................+++
...............+++
writing new private key to 'certs/ca.key'
-----
Importing keystore certs/server.p12 to certs/server.keystore.jks...
Entry for alias 1 successfully imported.
Import command completed:  1 entries successfully imported, 0 entries failed or cancelled
Certificate was added to keystore
Creating network "kafka-docker_default" with the default driver
Creating zookeeper ... done
Creating kafka     ... done
```
This command will generate a self-signed certificate (and other necessary files) which will be located in kafka-docker/certs folder.
And the command will up start a docker container with a Kafka inside.
The most important generate file is ca.crt. It is a self-signed SSL certificate. Every participant must have it
on the same machine where dc4bc_d is running and must provide path to the file in thee start command of dc4bc_d.


#### DKG

Generate keys for your node:
```
$ ./dc4bc_d gen_keys --username john_doe --key_store_dbdsn /tmp/dc4bc_john_doe_key_store
```
Start the node (note the `--storage_topic` flag — use a fresh topic for cleaner test runs):
```
$ ./dc4bc_d start --username john_doe --key_store_dbdsn /tmp/dc4bc_john_doe_key_store --listen_addr localhost:8080 --state_dbdsn /tmp/dc4bc_john_doe_state --storage_dbdsn 94.130.57.249:9093 --producer_credentials producer:producerpass --consumer_credentials consumer:consumerpass --kafka_truststore_path ./ca.crt --storage_topic test_topic
```
Start the airgapped machine:
```
$ ./dc4bc_airgapped --db_path /tmp/dc4bc_john_doe_airgapped_state --password_expiration 10m
```
Print your communication public key and encryption public key and save it somewhere for later use:
``` 
$ ./dc4bc_cli get_pubkey --listen_addr localhost:8080
EcVs+nTi4iFERVeBHUPePDmvknBx95co7csKj0sZNuo=
# Inside the airgapped shell:
>>> show_dkg_pubkey
sN7XbnvZCRtg650dVCCpPK/hQ/rMTSlxrdnvzJ75zV4W/Uzk9suvjNPtyRt7PDXLDTGNimn+4X/FcJj2K6vDdgqOrr9BHwMqJXnQykcv3IV0ggIUjpMMgdbQ+0iSseyq
```

Now you want to start the DKG procedure. This tells the node to send an InitDKG message that proposes to run DKG with parameters which locate in a start_dkg_propose.json file.
```
$ ./dc4bc_cli start_dkg /path/to/start_dkg_propose.json --listen_addr localhost:8080
```
Example of start_dkg_propose.json file structure:
```
{
  "SigningThreshold": 2,
  "Participants": [
    {
    "Username": "john_doe",
    "PubKey": "EcVs+nTi4iFERVeBHUPePDmvknBx95co7csKj0sZNuo=",
    "DkgPubKey": "sN7XbnvZCRtg650dVCCpPK/hQ/rMTSlxrdnvzJ75zV4W/Uzk9suvjNPtyRt7PDXLDTGNimn+4X/FcJj2K6vDdgqOrr9BHwMqJXnQykcv3IV0ggIUjpMMgdbQ+0iSseyq"
    },
    {
      "Username": "jane_doe",
      "PubKey": "cHVia2V5Mg==",
      "DkgPubKey": "ZGtnX3B1YmtleV8y"
    }
  ]
}
```

The message will be consumed by your node:
```
[john_doe] starting to poll messages from append-only log...
[john_doe] Starting HTTP server on address: localhost:8080
[john_doe] Handling message with offset 0, type event_sig_proposal_init
[john_doe] message event_sig_proposal_init done successfully from john_doe
[john_doe] Successfully processed message with offset 0, type event_sig_proposal_init
```

Now you have a pending operation in your operation pool. Get the list of pending operations:
```
$ ./dc4bc_cli get_operations --listen_addr localhost:8080
DKG round ID: 3086f09822d7ba4bfb9af14c12d2c8ef
Operation ID: 30fa9c21-b79f-4a53-a84b-e7ad574c1a51
Description: confirm participation in the new DKG round
Hash of the proposing DKG message - a60bd47a831cd58a96bdd4381ee15afc
-----------------------------------------------------
```

You can check the hash of the proposing DKG message:
```
./dc4bc_cli get_start_dkg_file_hash start_dkg_propose.json
a60bd47a831cd58a96bdd4381ee15afc
```
The command returns a hash of the proposing message. If it is not equal to the hash from the list of pending operations, that means the person who proposed to start the DKG round changed the parameters that you agreed on the Conferce Call.

Copy the Operation ID and make the node produce a QR-code for it:
```
$ ./dc4bc_cli get_operation_qr 6d98f39d-1b24-49ce-8473-4f5d934ab2dc --listen_addr localhost:8080
QR code was saved to: /tmp/dc4bc_qr_6d98f39d-1b24-49ce-8473-4f5d934ab2dc-0.gif
```

A single operation might be split into several QR-codes, which will be located in a single GIF file. Open the GIF-animation in any gif viewer and take a video of it:
```
open -a /Applications/Safari.app/ /tmp/dc4bc_qr_c76396a6-fcd8-4dd2-a85c-085b8dc91494-response.gif
```

After that, you need to scan the GIF. To do that, you need to open the `./qr_reader_bundle.html` in your Web browser, allow the page to use your camera and demonstrate the recorded video to the camera. After the GIF is scanned, you'll see the operation JSON. Click on that JSON, and it will be saved to your Downloads folder.

Now go to `dc4bc_airgapped` prompt and enter the path to the file that contains the Operation JSON:

```
>>> read_operation
> Enter the path to Operation JSON file: ~/Downloads/operation.json
2020/11/27 16:47:22 QR code was saved to: /tmp/dc4bc_qr_ce30c6a2-f5d6-43a1-ac7f-0a63b01ca6f8-response.gif
An operation in the read QR code handled successfully, a result operation saved by chunks in following qr codes:
Operation's chunk: /tmp/dc4bc_qr_ce30c6a2-f5d6-43a1-ac7f-0a63b01ca6f8-response.gif
```

Open the response QR-gif in any gif viewer and take a video of it. Refresh the `./qr_reader_bundle/index.html` page in your web browser and scan the GIF. You may want to give the downloaded file a new name, e.g., `operation_response.json`.

Then go to the node and run:
```
$ ./dc4bc_cli read_operation_result --listen_addr localhost:8080 ~/Downloads/operation_response.json
```

After reading the response, a message is send to the message board. When all participants perform the necessary operations, the node will proceed to the next step:
```
[john_doe] message event_sig_proposal_confirm_by_participant done successfully from john_doe
```
Further actions are repetitive. Check for new pending operations:
```
$ ./dc4bc_cli get_operations --listen_addr localhost:8080
```

Then feed them to `dc4bc_airgapped`, then pass the responses to the client, then wait for new operations, etc. After some back and forth you'll see the node tell you that DKG is finished (`event_dkg_master_key_confirm_received`):
```
[john_doe] State stage_signing_idle does not require an operation
[john_doe] Successfully processed message with offset 10, type event_dkg_master_key_confirm_received
``` 

#### Signature

Now we have to collectively sign a message. Some participant will run the command that sends an invitation to the message board:

```
# Inside dc4bc_airgapped prompt:
$ >>> show_finished_dkg
AABB10CABB10
$ echo "the message to sign" > data.txt
$ ./dc4bc_cli sign_data AABB10CABB10 data.txt --listen_addr localhost:8080
```  
Further actions are repetitive and are similar to the DKG procedure. Check for new pending operations, feed them to `dc4bc_airgapped`, pass the responses to the client, then wait for new operations, etc. After some back and forth you'll see the node tell you that the signature is ready:
```
[john_doe] Handling message with offset 40, type signature_reconstructed
Successfully processed message with offset 40, type signature_reconstructed
```

Now you have the full reconstructed signature. 
```
./dc4bc_cli get_signatures AABB10CABB10
Signing ID: 909b7660-ccc4-45c4-9201-e30015a69425
	DKG round ID: AABB10CABB10
	Participant: john_doe
	Reconstructed signature for the data: tK+3CV2CI0flgwWLuhrZA5eaFfuJIvpLAc6CbAy5XBuRpzuCkjOZLCU6z1SvlwQIBJp5dAVa2rtbSy1jl98YtidujVWeUDNUz+kRl2C1C1BeLG5JvzQxhgr2dDxq0thu
```
It'll show you a list of broadcasted reconstructed signatures for a given DKG round.

You can verify any signature by executing `verify_signature` command inside the airgapped prompt:
```
>>> verify_signature
> Enter the DKGRoundIdentifier: AABB10CABB10
> Enter the BLS signature: tK+3CV2CI0flgwWLuhrZA5eaFfuJIvpLAc6CbAy5XBuRpzuCkjOZLCU6z1SvlwQIBJp5dAVa2rtbSy1jl98YtidujVWeUDNUz+kRl2C1C1BeLG5JvzQxhgr2dDxq0thu
> Enter the message which was signed (base64): dGhlIG1lc3NhZ2UgdG8gc2lnbgo=
Signature is correct!
```
