package iot

import "sync"

// This is really a metaverse thing but we need to store a cache
// because building whole tree on the client is too slow. So we store the whole tree in a cache and then send it to the client when they connect.

// What happens when they get HUGE?
// A: Never fear success.

var WorldName2ChildbitsWholeString = make(map[string][]byte)

// a timestamp of when the last time we set the childbits for this world
var WorldName2lastset = make(map[string]int64)

var WorldName2ChildbitsLock = sync.Mutex{}

// nevermind. I wanted to live in a trailer but CP talked me out of it.
// var ChildbitsWholeString = make([]byte, 0)
