
## Tickets aka tokens used for access defined here.

See: `type KnotFreePayload struct`

and `MakeTicket(data *KnotFreePayload, privateKey []byte) ([]byte, error)`

and `func VerifyTicket(ticket []byte, publicKey []byte) (*KnotFreePayload, bool)`

