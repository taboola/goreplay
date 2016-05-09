package rawSocket

const queueSize uint64 = 8192
const indexMask uint64 = queueSize - 1

type PacketQueue struct {
    padding1 [8]uint64
    lastCommittedIndex uint64
    padding2 [8]uint64
    nextFreeIndex uint64
    padding3 [8]uint64
    readerIndex uint64
    padding4 [8]uint64
    contents [queueSize]*TCPPacket
    padding5 [8]uint64
}

func (self *PacketQueue) Write(value *TCPPacket) {
    var myIndex = atomic.AddUint64(&self.nextFreeIndex, 1) - 1
    //Wait for reader to catch up, so we don't clobber a slot which it is (or will be) reading
    for myIndex > (self.readerIndex + queueSize - 2) {
        runtime.Gosched()
    }
    //Write the item into it's slot
    self.contents[myIndex & indexMask] = value
    //Increment the lastCommittedIndex so the item is available for reading
    for !atomic.CompareAndSwapUint64(&self.lastCommittedIndex, myIndex - 1, myIndex) {
        runtime.Gosched()
    }
}

func (self *PacketQueue) Read() *TCPPacket {
    var myIndex = atomic.AddUint64(&self.readerIndex, 1) - 1
    //If reader has out-run writer, wait for a value to be committed
    for myIndex > self.lastCommittedIndex {
        runtime.Gosched()
    }
    return self.contents[myIndex & indexMask]
}


type MessageQueue struct {
    padding1 [8]uint64
    lastCommittedIndex uint64
    padding2 [8]uint64
    nextFreeIndex uint64
    padding3 [8]uint64
    readerIndex uint64
    padding4 [8]uint64
    contents [queueSize]*TCPMessage
    padding5 [8]uint64
}

func (self *MessageQueue) Write(value *TCPMessage) {
    var myIndex = atomic.AddUint64(&self.nextFreeIndex, 1) - 1
    //Wait for reader to catch up, so we don't clobber a slot which it is (or will be) reading
    for myIndex > (self.readerIndex + queueSize - 2) {
        runtime.Gosched()
    }
    //Write the item into it's slot
    self.contents[myIndex & indexMask] = value
    //Increment the lastCommittedIndex so the item is available for reading
    for !atomic.CompareAndSwapUint64(&self.lastCommittedIndex, myIndex - 1, myIndex) {
        runtime.Gosched()
    }
}

func (self *MessageQueue) Read() *TCPMessage {
    var myIndex = atomic.AddUint64(&self.readerIndex, 1) - 1
    //If reader has out-run writer, wait for a value to be committed
    for myIndex > self.lastCommittedIndex {
        runtime.Gosched()
    }
    return self.contents[myIndex & indexMask]
}