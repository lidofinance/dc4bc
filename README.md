# dc4bc: distributed custody for the beacon chain

The goal of ths project is to make a simple, secure framework to generate and use threshold signatures for infrequent financial transactions over Ethereum 2.0 Beacon Chain (BLS on BLS12-381 curve). dc4bc only deals with key generation and the signature process with all the user-side logic offloaded to applications using dc4bc as a service or an API.

For a better key management, we presume that, when used in production, private encryption keys and threshold signature related secrets reside in an airgapped machine or an HSM. For a better auditablity and testability, network protocol logic is implemented as a set of finite state machines that change state deterministically in response to a a stream of outside events. 

The main and, for now, only network communication primitive we use is a shared bulletin board in form of an authenticated append-only log. Different implementations of that log could be a shared file (for local development or testing), a trusted network service (e.g. Amazon S3 bucket), a federated blockchain between protocol participants or a public blockchain. 

# How to test this code?

Run the command below to run unit-tests:

```
make test-short
```

# How to run this code?

Please refer to [this page](HowTo.md) for a complete guide for running the minimal application testnet.

# Repository description

* `./airgapped` The Airgapped machine source code. All encryption- and DKG-related code can be found in this package;
* `./client` The Client source code. The Client can poll messages from the message board. It also sets up a local http-server to process incoming requests (e.g., "please start a new DKG round");
* `./cmd` Command line interfaces for the Airgapped machine and the Client. All entry points to dc4bc apps can be found here;
* `./dkg` This package is more of a library for maintaining all active DKG instances and data;
* `./fsm` The FSM source code. The FSM decides when we are ready to move to the next step during DKG and signing;
* `./qr` A library for handling QR codes that encode pending Operations (which are used for communication between The Client, and the Airgapped machine); 
* `./storage` Two Bulletin Board implementations: File storage for local debugging and Kafka storage for real-world scenarios.

# Related repositories

* [kyber-bls12381](https://github.com/depools/kyber-bls12381) BLS threshold signature library based on kyber's BLS threshold signatures library
* (bls12-381)[https://github.com/depools/bls12-381] high speed BLS12-381 implementation in go, based on kyber's one
* (kyber)[https://github.com/corestario/kyber/] dkg library, fork of DEDIS' kyber library

## Moving parts

### Participants 
N participants, having a hot (connected to the network) node and a cold (airgapped) node. Participants all have two pair of keys for digital signatures: one for hot node and one for airgapped. PubHotKey_i, PrivHotKey_i, PubColdKey_i, PrivColdKey_i respectively for Participant_i. Each participant also have a secret seed used to generate DKG messages: given the same seed and the same inbound DKG messages participant's outbound messages are deterministic.

Hot keys are stored on the network-connected node, cold keys and a seed are stored on an airgapped node.

### Conference call

It's presumed participants can use a separate secure communication channel (let's call it Conference Call) to establish initial parameters: the set of participants, their identities and public authentification keys, the nature and connection parameters of a bulletin board and so on.


### Bulletin Board

The core communication/storage primitive for dc4bc is a bulletin board - a simple authenticated append-only log that can be accesed by all the participants and allows posting authentificated messages and polling for posted messages. We need BB to have two functions:
- post(message, signature)
- getMessages(offset = 0)
  - returns a list of all messages posted after the first <offset> one

This allows us to establish communication primitives:

- broadcast(message) by Participant_i:
    post(message, signature(message, PrivHotKey_i))
- private_message(message, Participant_j):
    encrypted_message = { "to" : Participant_j, "message": encrypt(message, PubColdKey_j)}
    broadcast(encrypted_message)
    
Encryption is done using AES526-GCM + AEAD.

Bulletin board can be constructed using a trusted centralized service a-la kafka queue (implemented), using public blockchain, or using a consensus between participants to establish a private blockchain. Anyway, it should be abstracted away in the client and signer both and easily switchable.

Bulletin board is only available on a hot node.

### Secure Channel

There is a secure comminication channel between a hot node and a cold node between each participant. We expect it to be a QR-code based asynchronous messaging protocol, but it can be something more complicated eventually, e.g. USB connection to the HSM. It's got two primitive functions:
- h2c_message(message) - send a message from hot node to cold node, returns message hash
- await_c2h_reply(hash(message)) - wait for reply from cold node


## DKG Process

1. Using a Conference Call, participants establish: the set of participants, public keys for authentfication and encryption, the nature and connection parameters of a bulletin board, step timeouts, threshold number.
2. Any participant broadcasts a DKG Startup Message, that contains the set of participants, and public keys for authentfication and encryption. Hash of that message later is used as a unique id of a DKG (used in messages to differentiate between multiple parallel DKGs if needed).
3. All participants broadcast their agreement to participate in this particular DKG within the agreed upon step timeout.
4. When all participants agree, every participant asks a cold node to publish a commit:
   1. message_hash = h2c_message(<start DKG with DKG_hash xxx, number of participants X, threshold Y>)
   2. broadcast(await_c2h_reply(message_hash))
5. When all participants publish a commit, every participant:
   1. h2c_message(<all commits>)
   2. message_hash = h2c_message(<send deals>)
   3. deals = await_c2h_reply(message_hash)
   4. for participant in participants:
      1. direct_message(participant, deal[participant])
6. When a pariticipant has recieved all the deals:
   1. They reconstruct the public key from the deals and broadcast it
7. If everyone broadcasts the same reconstructed public key, DKG completed successfully

If at any point something goes wrong (timeout reached, the deal is invalid, public key is not recinstucted equally, some of participants complain using a Conference Call) the DKG is aborted.

## Signature process
1. Any paricipant broadcast a message to sign upon.
2. All other participants signal their willingness to sign by broadcasting agreemen to sign that message.
3. When enough (>= threshold) participants broadcasted an agreement, every participant:
   1. message_hash = h2c_message(<send a partial signature for message "message" for threshold public key "key">)
   2. broadcast(await_c2h_reply(message_hash))
4. When enough (>= threshold) participants broadcasted a partial signature, threshold signature is reconstructed.
5. Someone broadcasts a partial signature.

If not enough participants signal their willingness to sign within a timeout or signal their rejection to sign, signature process is aborted.

We organize logic in the hot node as a set of simple state machines that change state only by external trigger, such as CLI command, message from cold node, or a new message on Bulletin Board. That way it can be easily tested and audited.
