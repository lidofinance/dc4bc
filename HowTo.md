# Step-by-step guide

Clone the project repository:
```
git clone git@github.com:depools/dc4bc.git
```

#### Installation (Linux)

First install the Go toolchain:
```
wget https://golang.org/dl/go1.15.2.linux-amd64.tar.gz
tar -C /usr/local -xzf go1.15.2.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```

When Go is set up, install OpenCV:
```
git clone https://github.com/hybridgroup/gocv.git
cd gocv
make install
```

If it works correctly, at the end of the entire process, the following message should be displayed:
```
gocv version: 0.22.0
opencv lib version: 4.4.0
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

When Go is set up, install OpenCV:
```
brew uninstall opencv
brew install opencv
brew install pkgconfig
```

Then build the project binaries:
```
# Go to the cloned repository.
cd dc4bc
make build-darwin
```

If you see unexpected output like this:
```
ld: warning: directory not found for option '-L/usr/local/Cellar/opencv/4.4.0/lib'
```

Try to run:
```
export CGO_CXXFLAGS="--std=c++11"
export CGO_CPPFLAGS="-I/usr/local/Cellar/opencv/4.4.0/include"
export CGO_LDFLAGS="-L/usr/local/Cellar/opencv/4.4.0/lib -lopencv_stitching -lopencv_superres -lopencv_videostab -lopencv_aruco -lopencv_bgsegm -lopencv_bioinspired -lopencv_ccalib -lopencv_dnn_objdetect -lopencv_dpm -lopencv_face -lopencv_photo -lopencv_fuzzy -lopencv_hfs -lopencv_img_hash -lopencv_line_descriptor -lopencv_optflow -lopencv_reg -lopencv_rgbd -lopencv_saliency -lopencv_stereo -lopencv_structured_light -lopencv_phase_unwrapping -lopencv_surface_matching -lopencv_tracking -lopencv_datasets -lopencv_dnn -lopencv_plot -lopencv_xfeatures2d -lopencv_shape -lopencv_video -lopencv_ml -lopencv_ximgproc -lopencv_calib3d -lopencv_features2d -lopencv_highgui -lopencv_videoio -lopencv_flann -lopencv_xobjdetect -lopencv_imgcodecs -lopencv_objdetect -lopencv_xphoto -lopencv_imgproc -lopencv_core"
```
Now try to build the project again.

#### DKG

Generate keys for your node:
```
$ ./dc4bc_d gen_keys --username john_doe --key_store_dbdsn /tmp/dc4bc_john_doe_key_store
```
Start the node:
```
$ ./dc4bc_d start --username john_doe --key_store_dbdsn /tmp/dc4bc_john_doe_key_store --listen_addr localhost:8080 --state_dbdsn /tmp/dc4bc_john_doe_state --storage_dbdsn 94.130.57.249:9092
```
Start the airgapped machine:
```
$ ./dc4bc_airgapped /tmp/dc4bc_john_doe_airgapped_state
```
Print your communication public key and encryption public key and save it somewhere for later use:
``` 
$ ./dc4bc_cli get_pubkey --listen_addr localhost:8080
EcVs+nTi4iFERVeBHUPePDmvknBx95co7csKj0sZNuo=
# Inside the airgapped shell:
>>> show_dkg_pub_key
sN7XbnvZCRtg650dVCCpPK/hQ/rMTSlxrdnvzJ75zV4W/Uzk9suvjNPtyRt7PDXLDTGNimn+4X/FcJj2K6vDdgqOrr9BHwMqJXnQykcv3IV0ggIUjpMMgdbQ+0iSseyq
```

Now you want to start the DKG procedure. This tells the node to send an InitDKG message that proposes to run DKG for 2 participants with `threshold=2`. You will be prompted to enter some required information about the suggested participants:
```
$ ./dc4bc_cli start_dkg 2 2 --listen_addr localhost:8080
Enter a necessary data for participant 0:
Enter address: e0d8083f8a2d18f310bfbdc9649a83664470f46053ab53c105a054b08f9eff85
Enter pubkey (base64): 4NgIP4otGPMQv73JZJqDZkRw9GBTq1PBBaBUsI+e/4U=
Enter DKGPubKey (base64): sN7XbnvZCRtg650dVCCpPK/hQ/rMTSlxrdnvzJ75zV4W/Uzk9suvjNPtyRt7PDXLDTGNimn+4X/FcJj2K6vDdgqOrr9BHwMqJXnQykcv3IV0ggIUjpMMgdbQ+0iSseyq
Enter a necessary data for participant 1:
Enter address: 11c56cfa74e2e221444557811d43de3c39af927071f79728edcb0a8f4b1936ea
Enter pubkey (base64): EcVs+nTi4iFERVeBHUPePDmvknBx95co7csKj0sZNuo=
Enter DKGPubKey (base64): kJbOTZSwOKWYfg1KD/VxfRDEfk7kSgMzYiALJaLn2HJ08x5kIJWqkzFi/Z0B3ZEgBJROOybWPMVnQOpQ/DQwxYbxa6kgOPPBnY5WshX14vkgAtv+gE062rWLtFVBqZI+
```

The message will be consumed by your node:
```
[john_doe] starting to poll messages from append-only log...
[john_doe] Starting HTTP server on address: localhost:8080
[john_doe] Handling message with offset 0, type event_sig_proposal_init
[john_doe] message event_sig_proposal_init done successfully from e0d8083f8a2d18f310bfbdc9649a83664470f46053ab53c105a054b08f9eff85
[john_doe] Successfully processed message with offset 0, type event_sig_proposal_init
```

Now you have a pending operation in your operation pool. Get the list of pending operations:
```
$ ./dc4bc_cli get_operations --listen_addr localhost:8080
Operation ID: 6d98f39d-1b24-49ce-8473-4f5d934ab2dc
Operation: {"ID":"6d98f39d-1b24-49ce-8473-4f5d934ab2dc","Type":"state_sig_proposal_await_participants_confirmations","Payload":"W3siUGFydGljaXBhbnRJZCI6MCwiQWRkciI6ImUwZDgwODNmOGEyZDE4ZjMxMGJmYmRjOTY0OWE4MzY2NDQ3MGY0NjA1M2FiNTNjMTA1YTA1NGIwOGY5ZWZmODVcbiIsIlRocmVzaG9sZCI6Mn0seyJQYXJ0aWNpcGFudElkIjoxLCJBZGRyIjoiMTFjNTZjZmE3NGUyZTIyMTQ0NDU1NzgxMWQ0M2RlM2MzOWFmOTI3MDcxZjc5NzI4ZWRjYjBhOGY0YjE5MzZlYVxuIiwiVGhyZXNob2xkIjoyfV0=","ResultMsgs":null,"CreatedAt":"2020-09-11T14:28:54.343122+03:00","DKGIdentifier":"191fb020fd30edd891b066f72e5a5e3a","To":"","Event":""}
-----------------------------------------------------
```

Copy the Operation ID and make the node produce a QR-code for it:
```
$ ./dc4bc_cli get_operation_qr 6d98f39d-1b24-49ce-8473-4f5d934ab2dc --listen_addr localhost:8080
List of paths to QR codes for operation 6d98f39d-1b24-49ce-8473-4f5d934ab2dc:
0) QR code: /tmp/dc4bc_qr_6d98f39d-1b24-49ce-8473-4f5d934ab2dc-0
1) QR code: /tmp/dc4bc_qr_6d98f39d-1b24-49ce-8473-4f5d934ab2dc-1
```

A single operation might be split into several QR-codes. Open the first QR code in your default image viewer and take a photo of it:
```
open /tmp/dc4bc_qr_6d98f39d-1b24-49ce-8473-4f5d934ab2dc-0
```

Now go to `dc4bc_airgapped` prompt and enter:

```
>>> read_qr
```

A new window will be opened showing what your laptop's camera sees. Place the photo of the QR from the previous step in front of the camera and wait for the airgapped machine to scan it. You will have to scan all operation QR codes that were produced by the node.

After you've scanned all QR codes, you will be shown the path to the QR code that contains the response of `dc4bc_airgapped`. Note that it may be split into several chunks:
```
An operation in the readed QR code handled successfully, a result operation saved by chunks in following qr codes:
Operation's chunk #0: result_qr_codes/state_sig_proposal_await_participants_confirmations_de09e754-3bc8-4e67-9651-dcdba3316dba_-0.png
Operation's chunk #1: result_qr_codes/state_sig_proposal_await_participants_confirmations_de09e754-3bc8-4e67-9651-dcdba3316dba_-1.png
Operation's chunk #2: result_qr_codes/state_sig_proposal_await_participants_confirmations_de09e754-3bc8-4e67-9651-dcdba3316dba_-2.png
Operation's chunk #3: result_qr_codes/state_sig_proposal_await_participants_confirmations_de09e754-3bc8-4e67-9651-dcdba3316dba_-3.png
Operation's chunk #4: result_qr_codes/state_sig_proposal_await_participants_confirmations_de09e754-3bc8-4e67-9651-dcdba3316dba_-4.png
Operation's chunk #5: result_qr_codes/state_sig_proposal_await_participants_confirmations_de09e754-3bc8-4e67-9651-dcdba3316dba_-5.png
Operation's chunk #6: result_qr_codes/state_sig_proposal_await_participants_confirmations_de09e754-3bc8-4e67-9651-dcdba3316dba_-6.png
```
Open the first response QR code in your default image viewer and take a photo of it. Then go to the node and run:
```
$ ./dc4bc_cli read_from_camera  --listen_addr localhost:8080
```

The procedure is the same as with `dc4bc_airgapped`: scan all QR codes until you see a success message:
```
Operation successfully scanned
```

After scanning the response, a message is send to the message board. When all participants perform the necessary operations, the node will proceed to the next step:
```
[john_doe] message event_sig_proposal_confirm_by_participant done successfully from b8c083cd717b9958e141be5956bab1e463a7a0d85e4fe8924833601d43d671c4
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
$ echo "the message to sign" | base64
dGhlIG1lc3NhZ2UgdG8gc2lnbgo=
$ ./dc4bc_cli sign_data AABB10CABB10 dGhlIG1lc3NhZ2UgdG8gc2lnbgo= --listen_addr localhost:8080
```  
Further actions are repetitive and are similar to the DKG procedure. Check for new pending operations, feed them to `dc4bc_airgapped`, pass the responses to the client, then wait for new operations, etc. After some back and forth you'll see the node tell you that the signature is ready:
```
[john_doe] Handling operation state_signing_partial_signs_collected in airgapped
[176 141 250 6 188 93 29 163 218 11 179 113 24 74 95 62 89 163 219 249 135 53 26 212 55 134 143 107 117 216 112 85 163 6 117 153 161 171 235 145 198 253 60 53 22 3 36 84]
```

Now you have the full reconstructed signature. 