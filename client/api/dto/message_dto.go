package dto

type MessageDTO struct {
	ID            string
	DkgRoundID    string
	Offset        uint64
	Event         string
	Data          []byte
	Signature     []byte
	SenderAddr    string
	RecipientAddr string
}
