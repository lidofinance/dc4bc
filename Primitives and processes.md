## DKG

### Conference call

It's presumed participants can use a separate secure communication channel (let's call it Conference Call) to establish initial parameters: the set of participants, their identities and public authentification keys, the nature and connection parameters of a bulletin board and so on.


### Participants 
N participants, having a hot (connected to the network) node and a cold (airgapped) node. Participants all have two pair of keys: auth keys and encryption keys. PubAuthKey_i, PrivAuthKey_i, PubEncKey_i, PrivEncKey_i respectively Participant_i. Each participant also have a secret seed used to generate DKG messages.

Auth keys are stored on the hot node, encryption keys and a seed are stored on a cold node.

### Bulletin Board

The core communication/storage primitive for dc4bc is a bulletin board - a simple append-only log that can be accesed by all the participants and allows posting authentificated messages and polling for posted messages. We need BB to have two functions:
- post(message, signature)
- getMessages(offset = 0)
  - returns a list of all messages posted after the first <offset> one

This allows us to establish communication primitives:

- broadcast(message) by Participant_i:
    post(message, signature(message, PrivAuthKey_i))
- private_message(message, Participant_j):
    encrypted_message = { "to" : Participant_j, "message": encrypt(message, PubEncKey_j)}
    broadcast(encrypted_message)

Bulletin board can be constructed using a trusted centralized service a-la github/amazon, using public blockchain, or using a consensus between participants to establish a private blockchain. Anyway, it should be abstracted away in the client and signer both and easily switchable.

Bulletin board is only available on a hot node.

### Secure Channel

There is a secure comminication channel between a hot node and a cold node between each participant. We expect it to be a dead simple QR-code based asynchronous messaging protocol, but it can be something more complicated eventually, e.g. USB connection to the HSM. It's got two primitive functions:
- h2c_message(message) - send a message from hot node to cold node, returns message hash
- await_c2h_reply(hash(message)) - wait for reply from cold node


## DKG Process

1. Using a Conference Call, participants establish: the set of participants, public keys for authentfication and encryption, the nature and connection parameters of a bulletin board, step timeouts, threshold number.
2. Any participant broadcast a DKG Startup Message, that contains the set of participants, and public keys for authentfication and encryption. Hash of that message later is used as a unique id of a DKG (used in messages to differentiate between multiple parallel DKGs if needed).
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

If not enough participants signal their willingness to sign within a timeout or signal their rejection to sign, 