# Step-by-step guide
0. Install Docker: https://docs.docker.com/get-docker/
1. Clone the project repository:  
    ```
    git clone git@github.com:lidofinance/dc4bc.git
    ```
2. Run the Client node inside a Docker container. Note that the `DATA_DIR` environment variable specifies the directory _on your host machine_ where persistent state will be kept (you can stop and run the container, and the data will be still there). Your keys will be generated automatically if this is the first run (i.e., if the `DATA_DIR` does not contain previously generated keys):  
    ```
    $ DATA_DIR=/tmp/shared USERNAME=john_doe STORAGE_DBDSN=51.158.98.208:9093 STORAGE_TOPIC=test_topic make run-client-node
    <...>
    Successfully built 9fe3bbdf08e6
    Successfully tagged client_node:latest
    Keystore is not found, generating new keys
    keypair generated for user john_doe and saved to /go/src/shared/dc4bc_john_doe_key_store
    Started QR scanner. Go to http://localhost:9090/qr/index.html
    [john_doe] Client started to poll messages from append-only log
    [john_doe] Waiting for messages from append-only log...
    [john_doe] HTTP server started on address: localhost:8080
    ```
3. Open `./qr/index.html` in your Web browser. This will start the QR-scanning application; you may need to give it permissions to use your camera. 
4. Open a separate terminal and log into the Client node. All the necessary binaries will be accessible in current working directory:
    ```
    $ make run-client-node-bash
    root@df8a53006ea0:/go/src#
    ```
5. Next run the Airgapped Machine inside a Docker container. You will be logged right into the Airgapped Machine prompt:
    ```
    $ DATA_DIR=/tmp/shared PASSWORD_EXPIRATION=1000m USERNAME=john_doe make run-airgapped-machine
    <...>
    Successfully built 041987128b41
    Successfully tagged airgapped_machine:latest
    b4c7ab1eb89a7258d9937751052f2f813832fb9dbbf5c84e4d4b038864568c65
    2020/11/27 21:14:34 Base seed not initialized, generating a new one...
    2020/11/27 21:14:34 Successfully generated a new seed
    Enter encryption password:
    Confirm encryption password:
    Available commands:
    * read_operation - reads base64-encoded Operation, handles a decoded operation and returns the path to the GIF with operation's result
    * verify_signature - verifies a BLS signature of a message
    * change_configuration - changes a configuration variables (frames delay, chunk size, etc...)
    * help - shows available commands
    * show_dkg_pubkey - shows a dkg pub key
    * show_finished_dkg - shows a list of finished dkg rounds
    * replay_operations_log - replays the operation log for a given dkg round
    * drop_operations_log - drops the operation log for a given dkg round
    * exit - stops the machine
    Waiting for command...
    >>>
    ```
   Again, note that the `DATA_DIR` environment variable specifies the directory _on your host machine_ where persistent state will be kept (you can stop and run the container, and the data will be still there).

N.B.: that if you want to generate new keys or to manually start/stop the client node, you can use these commands inside the node container (although the preferred way is to clear the `DATA_DIR` and to start/stop the container respectively):
```
$ ./dc4bc_d gen_keys --username $USERNAME --key_store_dbdsn /tmp/dc4bc_$USERNAME_key_store
$ ./dc4bc_d start --username $USERNAME --key_store_dbdsn /tmp/dc4bc_$USERNAME_key_store --listen_addr localhost:8080 --state_dbdsn /tmp/dc4bc_john_doe_state --storage_dbdsn 51.158.98.208:9093 --producer_credentials producer:producerpass --consumer_credentials consumer:consumerpass --kafka_truststore_path ./ca.crt --storage_topic test_topic
```
$ 


#### DKG

Print your communication public key and encryption public key and save it somewhere for later use:
``` 
# Inside the Client node shell:
$ ./dc4bc_cli get_pubkey
EcVs+nTi4iFERVeBHUPePDmvknBx95co7csKj0sZNuo=

# Inside the Airgapped Machine shell:
>>> show_dkg_pubkey
tJVJWLQlHY2Jpo1CCgRTzq3LHU/rmPuobGGwxe6gCHgUrFKCOxgzfSYNRl3HsnFp
```

Now you want to start the DKG procedure. This tells the node to send an InitDKG message that proposes to run DKG with parameters which locate in a start_dkg_propose.json file.
```
# Inside the Client node shell:
$ ./dc4bc_cli start_dkg /path/to/start_dkg_propose.json
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
# Inside the Client node shell:
$ ./dc4bc_cli get_operations
DKG round ID: 3086f09822d7ba4bfb9af14c12d2c8ef
Operation ID: 30fa9c21-b79f-4a53-a84b-e7ad574c1a51
Description: confirm participation in the new DKG round
Hash of the proposing DKG message - a60bd47a831cd58a96bdd4381ee15afc
-----------------------------------------------------
```

You can check the hash of the proposing DKG message:
```
# Inside the Client node shell:
./dc4bc_cli get_start_dkg_file_hash start_dkg_propose.json
a60bd47a831cd58a96bdd4381ee15afc
```
The command returns a hash of the proposing message. If it is not equal to the hash from the list of pending operations, that means the person who proposed to start the DKG round changed the parameters that you agreed on the Conferce Call.

Copy the Operation ID and make the node produce a QR-code for it:
```
# Inside the Client node shell:
$ ./dc4bc_cli get_operation_qr 6d98f39d-1b24-49ce-8473-4f5d934ab2dc
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
$ ./dc4bc_cli read_operation_result ~/Downloads/operation_response.json
```

After reading the response, a message is send to the message board. When all participants perform the necessary operations, the node will proceed to the next step:
```
[john_doe] message event_sig_proposal_confirm_by_participant done successfully from john_doe
```
Further actions are repetitive. Check for new pending operations:
```
$ ./dc4bc_cli get_operations
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
