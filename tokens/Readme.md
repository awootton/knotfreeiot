
## Tokens used for access defined here.

See: `type KnotFreeTokenPayload struct`

and `MakeToken(data *KnotFreeTokenPayload, privateKey []byte) ([]byte, error)`

and `func VerifyToken(ticket []byte, publicKey []byte) (*KnotFreeTokenPayload, bool)`


