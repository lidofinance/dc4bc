package qr

import (
	"fmt"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/mattn/go-gtk/glib"
	"github.com/mattn/go-gtk/gtk"

	encoder "github.com/skip2/go-qrcode"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
)

const tmpImageFile = "/tmp/qr.png"

func ShowQR(data string) error {
	err := encoder.WriteFile(data, encoder.Medium, 512, tmpImageFile)
	if err != nil {
		return fmt.Errorf("failed to encode the data: %w", err)
	}

	showImage(tmpImageFile)

	return nil
}

func showImage(imageFile string) {
	gtk.Init(nil)
	window := gtk.NewWindow(gtk.WINDOW_TOPLEVEL)
	window.SetPosition(gtk.WIN_POS_CENTER)
	window.SetTitle("p2p.org QR Viewer")
	window.SetIconName("p2p.org QR Viewer")
	window.Connect("destroy", func(ctx *glib.CallbackContext) {
		gtk.MainQuit()
	})

	hbox := gtk.NewHBox(false, 1)
	hpaned := gtk.NewHPaned()
	hbox.Add(hpaned)
	frame1 := gtk.NewFrame("QR Code")
	framebox1 := gtk.NewHBox(false, 1)
	frame1.Add(framebox1)
	hpaned.Pack1(frame1, false, false)
	image := gtk.NewImageFromFile(imageFile)
	framebox1.Add(image)
	window.Add(hbox)
	imagePixBuffer := image.GetPixbuf()
	horizontalSize := imagePixBuffer.GetWidth()
	verticalSize := imagePixBuffer.GetHeight()

	window.SetSizeRequest(horizontalSize, verticalSize)
	window.ShowAll()
	gtk.Main()
}
