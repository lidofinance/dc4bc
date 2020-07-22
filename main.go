package main

import (
	_ "image/jpeg"
	"log"
	"os"
	"os/exec"

	"p2p.org/dc4bc/qr"

	_ "image/gif"
	_ "image/png"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

func main() {
	clearTerminal()
	var data = "Hello, world!"

	log.Println("A QR code will be shown on your screen.")
	log.Println("Please take a photo of the QR code with your smartphone.")
	log.Println("When you close the image, you will have 5 seconds to" +
		"scan the QR code with your laptop's camera.")
	err := qr.ShowQR(data)
	if err != nil {
		log.Fatalf("Failed to show QR code: %v", err)
	}

	var scannedData string
	for {
		clearTerminal()
		if err != nil {
			log.Printf("Failed to scan QR code: %v\n", err)
		}

		log.Println("Please center the photo of the QR-code in front" +
			"of your web-camera...")

		scannedData, err = qr.ReadQRFromCamera()
		if err == nil {
			break
		}
	}

	clearTerminal()
	log.Printf("QR code successfully scanned; the data is: %s\n", scannedData)
}

func clearTerminal() {
	c := exec.Command("clear")
	c.Stdout = os.Stdout
	_ = c.Run()
}
