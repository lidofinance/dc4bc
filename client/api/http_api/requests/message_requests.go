package requests

type MessageForm struct {
	ID            string `json:"id"`
	DkgRoundID    string `json:"dkg_round_id" validate:"attr=dkg_round_id,min=3"`
	Offset        uint64 `json:"offset"`
	Event         string `json:"event"`
	Data          []byte `json:"data"`
	Signature     []byte `json:"signature"`
	SenderAddr    string `json:"sender"`
	RecipientAddr string `json:"recipient"`
}
