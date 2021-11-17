# dc4bc threshold signature distributed key generation step-by-step guide

You'll generate key shards for a threshold signature potentially holding multimillion balance. We designed a pretty complicated software to handle that; it's a bit rough around the edges but serviceable. It's using a hot internet-connected machine for communication and an airgapped machine for handling secrets; we'd prefer to use hardware wallet for this but it wasn't possible on our timeline. 

## Large-scale steps:

0. Build or download software
1. Setup hot and airgapped machines, backup media 
2. Generate participant's keypairs
3. Publish your public keys and agree on threshold signature participant set
4. Initiate dkg ceremony
5. Confirm ceremony participation
6. Broadcast commits
7. Collect commits and broadcast deals
8. Collect deals and broadcast reconstructed public key
9. Do a test run of threshold signing a message
10. Check signature correctness

### Building software

Clone the project repository:
```
git clone https://github.com/lidofinance/dc4bc.git
```

#### Installation (Linux)

First install the Go toolchain:
```
curl -OL https://golang.org/dl/go1.15.2.linux-amd64.tar.gz
tar -C /usr/local -xzf go1.15.2.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```

Then build the project binaries:
```
# Go to the cloned repository.
cd dc4bc
make build
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
make build
```

### Downloading

Check out project releases tab in github and get the distribuition binaries for your system. Also clone the repository anyway, because you'll need the certificate file for kafka that is not a part of the releases files.

#### Setting up hot and airapped nodes

Following is a reasonably secure setup for an airgapped machine. It does not account for supply chain attacks (someone shipping you posined hardware) and wireless physical proximity 0day exploits but otherwise is reasonably good. With good the geographical and otherwise diversity of co-signers this should be good enough to make collusion the main practical avenue of attack.


Setup:

Hot node: linux or os x machine with webcam. Probably a laptop. Setup doesn't really matter as long as it's your machine (not shared or rented) and you're reasonably sure it's not compromised. Should have a reliable network connection during the ceremony. It stores communication keys that are do not protect any critical secrets.

Airgapped node: 

A laptop that can run Tails (linux distribution) and has a webcam. Preferably it has hardware switch for wireless connections or, even better, wireless chips removed. Shouldn't have wireless keyboard or mouse adapters plugged in either.

NB: If you know what you're doing set up an airgapped machine yourself or use one you already have. Tails live dvd/usb is not the best possible setup - it's just good enough in our opinion.

Backup media: paper wallet or another media you will use to backup bip39 word-based seed. You will keep this backup until withdrawals are enabled in eth2.

Plaintext media: 1+ gb usb drive or cd/dvd for non-secrets (executables and the like). If you choose USB drive, you'll be disposing of that particular drive by the end of a ceremony.

1. Make a bootable media for tails using https://tails.boum.org/install/index.en.html instructions. Live dvd is preferabe to a usb stick but usb stick is a valid option. Verification of an image signature per instructions on Tail's site is strongly recommended. 
2. Prepare a plaintext media: run a script `sh airapped_folder/setup.sh` to set up a folder with one on your hot node, then copy/burn that folder on the plaintext media. It contains airapped node binary, firefox distribution, qr code reader html file and deploy script to easily copy all that to tails distribution.
3. If your airgapped laptop allows it, switch the wireless hardware off. Boot into Tails, selecting "no network connection" as an additional option on starting (always select this option on an airgapped machine). From this point on to the end of ceremony the machine shouldn't ever connect to the network; if you can afford having it permanently airgapped forever - can by useful in crypto - do it.
4. Insert a plaintext media and run `sh ./deploy.sh` from there to copy all the needed files to your ephemeral home dir on tails.

Now you've got all paraphernalia set up and can proceed with the guide further.

### DKG

The goal of DKG is to produce a set of secrets, and those secrets can be potentially used for managing vast amounts of money. Threfore it is obvious that you would like your private key share to be generated and stored as securely as possible. To achieve the desired security level, you should have access to two computers: one for the Client node (with a web camera and with access to the Internet) and one for the Airgapped machine (just with a web camera).

### Generating keypairs

To start a DKG round, you should first generate two pairs of keys: one pair is for signing messages that will go to the Bulletin Board, and the other one will be used by the Airgapped Machine to encrypt private messages (as opposed to the messages that are broadcasted).

First, generate keys for your Client node:
```
$ ./dc4bc_d gen_keys --username <YOUR USERNAME> --key_store_dbdsn ./stores/dc4bc_<YOUR USERNAME>_key_store
```
Immediately backup the key store: these keys won't. be the ones to hold money, but if they are lost durin. the initial ceremony dkg round will have to be reasterted.

Then start the on the airgapped machine:
```
$ ./dc4bc_airgapped --db_path ./stores/dc4bc_<YOUR USERNAME>_airgapped_state --password_expiration 10m
```
* `--db_path` Specifies the directory in which the Aigapped machibne state will be stored. If the directory that you specified does not exist, the Airgapped machine will generate new keys for you on startup. *N.B.: It is very important not to put your Airgapped machine state to `/tmp` or to occasionally lose it. Please make sure that you keep your Airgapped machine state in a safe place and make a backup.*

Backup the generated bip39 seed on a paper wallet; if you need to restore it, use the `set_seed` command in the airgapped executable's console.

* `--password_expiration` Specifies the time in which you'll be able to use the Airgapped machine without re-entering your password. The Airgapped machine will ask you to create a new password during the first run. Make sure that the password is not lost.

After you have the keys, start the node:
```
$ ./dc4bc_d start --username <YOUR USERNAME> --key_store_dbdsn ./stores/dc4bc_<YOUR USERNAME>_key_store --state_dbdsn ./stores/dc4bc_<YOUR USERNAME>_state --listen_addr localhost:8080 --producer_credentials producer:producerpass --consumer_credentials consumer:consumerpass --kafka_truststore_path ./ca.crt --storage_dbdsn 51.158.98.208:9093 --storage_topic <DKG_TOPIC> --kafka_consumer_group <YOUR USERNAME>_group
```
* `--username` — This username will be used to identify you during DKG and signing
* `--key_store_dbdsn` — This is where the keys that are used for signing messages that will go to the Bulletin Board will be stored. Do not store these keys in `/tmp/` for production runs and make sure that you have a backup
* `--state_dbdsn` This is where your Client node's state (including the FSM state) will be kept. If you delete this directory, you will have to re-read the whole message board topic, which might result in odd states
* `--storage_dbdsn` This argument specifies the storage endpoint. This storage is going to be used by all participants to exchange messages
* `--storage_topic` Specifies the topic (a "directory" inside the storage) that you are going to use. Typically participants will agree on a new topic for each new signature or DKG round to avoid confusion
* `--kafka_consumer_group` Specifies your consumer group. This allows you to restart the Client and read the messages starting from the last one you saw.

Print your communication public key and encryption public key. *You will have to publish them during the [Conference call](https://github.com/lidofinance/dc4bc-conference-call) along with the `--username` that you specified during the Client node setup).*
```
$ ./dc4bc_cli get_pubkey --listen_addr localhost:8080
EcVs+nTi4iFERVeBHUPePDmvknBx95co7csKj0sZNuo=
# Inside the airgapped shell:
>>> show_dkg_pubkey
sN7XbnvZCRtg650dVCCpPK/hQ/rMTSlxrdnvzJ75zV4W/Uzk9suvjNPtyRt7PDXLDTGNimn+4X/FcJj2K6vDdgqOrr9BHwMqJXnQykcv3IV0ggIUjpMMgdbQ+0iSseyq
>>> generate_dkg_pubkey_json
A JSON file with DKG public key was saved to: /tmp/dc4bc_json_dkg_pub_key.json
```


**N.B.: You can start and stop both the Client node and the Airgapped machine any time you want given that the states are stored safely on your computer. When you restart the Airgapped machine, make sure that you run the `replay_operations_log` command exactly once before performing any actions — that will make the Airgapped machine replay the state and be ready for new actions. Please do not replay the log more than once during one Airgapped session, this might lead to undefined state.**

Now you want to start the DKG procedure. *This action must be done exactly once by only one of the participants. The participants must decide who will send the initial message collectively.* 

Tell the node to send an InitDKG message that proposes to run DKG with parameters which are located in a `start_dkg_propose.json` file. This file is created collectively during a [Conference call](https://github.com/lidofinance/dc4bc-conference-call) by the participants.
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

Now you have a pending operation in your operation pool. That is an operation to confirm participation: you'll be bringing dkg setup data to the airgapped machine and submitting the message that you will participate. 

Get the list of pending operations:
```
$ ./dc4bc_cli get_operations --listen_addr localhost:8080
Please, select operation:
-----------------------------------------------------
 1)             DKG round ID: c04f3d54718dfc801d1cbe86e3a265f5342ec2550f82c1c3152c36763af3b8f2
                Operation ID: 83e14a21c0116094630654d923ba9600
                Description: confirm participation in the new DKG round
-----------------------------------------------------
Select operation and press Enter. Ctrl+C for cancel

```

You can check the hash of the proposing DKG message:
```
./dc4bc_cli get_start_dkg_file_hash start_dkg_propose.json
a60bd47a831cd58a96bdd4381ee15afc
```
The command returns a hash of the proposing message. If it is not equal to the hash from the list of pending operations, that means the person who proposed to start the DKG round changed the parameters that you agreed on the Conferce Call.

Select an operation by typing its number and press Enter. This operation requires only a confirmation of user, so it's just sends a message to the append-only log.

When all participants confirm their participation in DKG round, the node will proceed to the next step:
```
[john_doe] message event_sig_proposal_confirm_by_participant done successfully from john_doe
```

Now you have a new operation:

```
$ ./dc4bc_cli get_operations --listen_addr localhost:8080
Please, select operation:
-----------------------------------------------------
 1) DKG round ID: c04f3d54718dfc801d1cbe86e3a265f5342ec2550f82c1c3152c36763af3b8f2
    Operation ID: df482be9eb1e50b0968a5daf7e52e073
    Description: send commits for the DKG round
-----------------------------------------------------
Select operation and press Enter. Ctrl+C for cancel

```

Select an operation to make the node produce a JSON file for it:
```
json file was saved to: /tmp/dkg_id_c04f3_step_1_send_commits_for_the_DKG_round_df482_request.json
```

Open the `./qr_reader_bundle/qr-tool.html` in your Web browser on the hot node machine and airgapped machine, allow the page to use your camera and demonstrate the recorded video to the camera.

Pull your JSON file to the encoder on the hot node machine and save the *.gif file.

Show this animation in the QR-tool on the airgapped machine and get JSON file.

Now go to `dc4bc_airgapped` prompt and enter the path to the file that contains the Operation JSON:

```
>>> read_operation
> Enter the path to Operation JSON file: /tmp/dkg_id_c04f3_step_1_send_commits_for_the_DKG_round_df482_request.json
Operation JSON was handled successfully, the result Operation JSON was saved to: /tmp/dkg_id_c04f3_step_1_send_commits_for_the_DKG_round_df482_response.json
```

Encode result JSON file to QR GIF on the airgapped machine and show animation on the hot node machine.

Then go to the node, decode GIF to JSON and run:
```
$ ./dc4bc_cli read_operation_result --listen_addr localhost:8080 /tmp/dkg_id_c04f3_step_1_send_commits_for_the_DKG_round_df482_response.json

```

When all participants perform the necessary operations, ßthe node will proceed to the next step:
```
[john_doe] message event_dkg_commit_confirm_received done successfully from john_doe
```

Next steps are:

- Broadcast commits - you'll be broadcasting a public derivative of your secret seed for the key shards that will be used to check that you don't try to cheat at DKG
- Collect commits and broadcast deals - you'll be collecting each other's commits and sending each participant a private message that will be used to construct your key shard
- Collect deals and broadcast reconstructed public key - you'll be using private messages from other people to generate your key shard

Further actions are repetitive:

For each step check for new pending operations:

```
$ ./dc4bc_cli get_operations --listen_addr localhost:8080
Please, select operation:
-----------------------------------------------------
1) DKG round ID: 3086f09822d7ba4bfb9af14c12d2c8ef
   Operation ID: 2f217f58-a94f-47d8-b871-f35a15275184
   Description: send commits for the DKG round
-----------------------------------------------------
Select operation and press Enter. Ctrl+C for cancel

```

Then feed them to `dc4bc_airgapped`, then pass the responses to the client, then wait for new operations, etc. After some back and forth you'll see the node tell you that DKG is finished (`event_dkg_master_key_confirm_received`):
```
[john_doe] State stage_signing_idle does not require an operation
[john_doe] Successfully processed message with offset 10, type event_dkg_master_key_confirm_received
```

Key generation ceremony is over. `exit` airgapped dkg tool prompt and backup your airapped machine db multiple times to different media. 

To check the backup run `dc4bc_airgapped -db_path /path/to/backup` and run `show_dkg_pubkey` command. If it works the backup is correct.

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

Now the ceremony is over. 

#### Reinitialize DKG

If you've lost all your states, communication keys, but your mnemonic for private DKG key is safe, it is possible to
reinitialize the whole DKG to recover DKG master key.

To do this, each participant must generate a new pair of communication keys (see above) and share a public one with other participants.
On your airgapped machine each participant must recover a private DKG key-pair:

```shell
>>> set_seed
> WARNING! this will overwrite your old seed, which might make DKGs you've done with it unusable.
> Only do this on a fresh db_path. Type 'ok' to  continue: ok
> Enter the BIP39 mnemonic for a random seed:
```
All participants must now share their public communication keys. Run the command below to get your public communication key:
```
$ ./dc4bc_cli get_pubkey --listen_addr localhost:8080
EcVs+nTi4iFERVeBHUPePDmvknBx95co7csKj0sZNuo=
```
Someone must put those keys to `keys.json` in the following format and send that file to all participants:
```
{
  "gergold": "8beWWNydtmEPISNXSC+Vp7U8nJrk23m9goW2hCX/eOo=",
  "svanevik": "r6cAAXor6iSy6nRipvLvfrNQe2BAVDQp8UN9Z+gZCNc=",
  "staking_facilities": "EYdeAeCJLNUXufn6SOFbdr0HWp3f0YwurHcVE2yGLvY=",
  "chorusone": "n/xcqib0t5bsyQ93Vfk4DXgahiEqrh4pEVKtWxH64UA=",
  "will": "qK3DvQ4S52/D8OLXDw3rue510A4GsEA7ZffVIuPoUyk=",
  "musalbas": "STFn4nporJXEPa+ftdztVo7zKa7Z4nG2eEV0jGJf2Mc=",
  "certus_one": "/QInqEyNBq0JZJuEQf1ZKLgQtTZJSdWJHr3KV5YAc0A=",
  "banteg": "ME06OxCRYZjz/sNv5mpBNRq2SHiVOAfmdaSyreSaEkk=",
  "k06a": "fK6y9yk06VNh8PZTwVQjEdyl40yta2aGlu0VXNYJfhM=",
  "rune": "v94jhyVQRRRaLiH4uxUljEeAC0uRzQERzCOsbU8lvYk=",
  "michwill": "FoiZGSPRFZTRZ98fTjjgeLs3fU8QRdXEadagcDW5zdY="
}
```

Someone then must use the ```dc4bc_dkg_reinitializer``` utility (available with the `darwin` or `linux` suffix on the release page, see above) to generate a reinit message for dc4bc_d. First you need to check the dump (downloaded at step 3 from `Initial setup`) and the `keys.json` checksum:
```
shasum dc4bc_async_ceremony_13_12_2020_dump.csv
b9934eeb7abf7a5563ad2ad06ede87ff58c89b0c  dc4bc_async_ceremony_13_12_2020_dump.csv
shasum keys.json
9c08507c073642c0e97efc87a685c908e871ef8a  keys.json
```
If the checksum is correct for all participants, run:
```shell
./dc4bc_dkg_reinitializer reinit -i dc4bc_async_ceremony_13_12_2020_dump.csv -o reinit.json -k keys.json --adapt_1_4_0 --skip-header
```
In this example the message will be saved to ```reinit.json``` file.
* `--adapt_1_4_0`: this flag patches the old append log so that it is compatible with the latest version. You can see the utility source code [here](https://github.com/lidofinance/dc4bc/blob/eb72f74e25d910fc70c4a77158fed07435d48d7c/client/client.go#L679);
* `-k keys.json`: new communication public keys from this file will be added to `reinit.json`.

**Note: all participants can run this command and check the `reinit.json` file checksum:**
```
./dc4bc_cli get_reinit_dkg_file_hash reinit.json
f65e4d87dce889df00ecebeed184ee601c23e531
```

Then someone must use ```reinit_dkg``` command in dc4bc_cli to send the message to the append-only log:
```shell
$ ./dc4bc_cli reinit_dkg reinit.json
```

The command will send the message to the append-only log, dc4bc_d process it and will return an operation that must be handled like in the previous steps (scan JSON, go to an airgapped machine, etc.).
```
$ ./dc4bc_cli get_operations
Please, select operation:
-----------------------------------------------------
 1)		DKG round ID: d62c6c478d39d4239c6c5ceb0aea6792
		Operation ID: 34799e2301ae794c0b4f5bc9886ed5fa
		Description: reinit DKG
		Hash of the reinit DKG message - f65e4d87dce889df00ecebeed184ee601c23e531
-----------------------------------------------------
Select operation and press Enter. Ctrl+C for cancel
```

There is a hash of the reinit DKG message in a reinitDKG operation and if it's not equal to the hash from ```get_reinit_dkg_file_hash``` command, that means that person who started the reinit process has changed some parameters.

After you have processed the operation in airgapped, you have your master DKG pubkey recovered, so you can sign new messages!
