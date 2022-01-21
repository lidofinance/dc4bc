package types

type BatchPartialSignatures map[string][][]byte

func (b BatchPartialSignatures) AddPartialSignature(messageID string, partialSignature []byte) {
	b[messageID] = append(b[messageID], partialSignature)
}

type ReconstructedSignature struct {
	File       string
	MessageID  string
	SrcPayload []byte
	Signature  []byte
	Username   string
	DKGRoundID string
}
