// Copyright 2019 Alan Tracey Wootton

package misc

// bound to network connections and the encoder and decoder would
// run in different processes.
// var network bytes.Buffer        // Stand-in for a network connection
// enc := gob.NewEncoder(&network) // Will write to network.
// dec := gob.NewDecoder(&network) // Will read from network.

// // Encode (send) some values.
// err := enc.Encode(P{3, 4, 5, "Pythagoras"})
// if err != nil {
// 	log.Fatal("encode error:", err)
// }
// err = enc.Encode(P{1782, 1841, 1922, "Treehouse"})
// if err != nil {
// 	log.Fatal("encode error:", err)
// }

// // Decode (receive) and print the values.
// var q Q
// err = dec.Decode(&q)
// if err != nil {
// 	log.Fatal("decode error 1:", err)
// }
// fmt.Printf("%q: {%d, %d}\n", q.Name, *q.X, *q.Y)
// err = dec.Decode(&q)
// if err != nil {
// 	log.Fatal("decode error 2:", err)
// }
// fmt.Printf("%q: {%d, %d}\n", q.Name, *q.X, *q.Y)

// func (p *TcpOverPubsubCmd) String() string {
// 	return "fff"
// }

// const (
// 	syn     = 1
// 	syn_ack = 2
// )

// var PacketNames = map[uint8]string{
// 	1: "syn",
// }

// func (p *TCPOverPubsubCmd) Write(w io.Writer) error {
// 	fmt.Println("writing ", *p)
// 	enc := gob.NewEncoder(w)
// 	err := enc.Encode(p)
// 	if err != nil {
// 		log.Fatal("encode error:", err)
// 	}
// 	return nil
// }

// func ReadCmd(r io.Reader) (CmdInterface, error) {
// 	return nil, nil
// }

// func NewCmd(packetType byte) CmdInterface {
// 	switch packetType {
// 	case syn:
// 		return &TcpOverPubsubCmd{}
// 	}
// 	return nil
// }

// func solder(from chan byte, to chan byte) {
// 	for {
// 		ch := <-from
// 		to <= ch
// 	}
// }

//var wire1 = make(chan byte, 5)
//
//  ByteChanReadWriter - implements io.Reader and io.Writer
// type ByteChanReadWriter struct {
// 	wire chan byte
// }

// func (me *ByteChanReadWriter) Write(p []byte) (n int, err error) {
// 	for _, ch := range p {
// 		//fmt.Print(string(ch))
// 		wire <- me.ch
// 	}
// 	return len(p), nil
// }

// func (*ByteChanReadWriter) Read(p []byte) (n int, err error) {
// 	for i := range p {
// 		ch := <-wire
// 		p[i] = ch
// 	}
// 	return len(p), nil
// }

// o := myRWthing{}
// 	n, err := o.Write([]byte("ohi"))
// 	_ = n
// 	_ = err

// cli2srvchan := make(chan TcpOverPubsubCmd)
// go func() {
// 	a := <-cli2srvchan
// 	fmt.Println("cmd=", a)
// }()
