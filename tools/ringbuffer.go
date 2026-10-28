package tools

import (
	"sync"
	"time"
)

type Packet struct {
	Timestamp time.Time
	Data      []byte
}

type ringBuffer struct {
	mu        sync.Mutex
	packets   []Packet
	size      int
	write     int
	read      int
	full      bool
	hasData   bool
	startTime time.Time
	endTime   time.Time
}

func newRingBuffer(maxPackets int) *ringBuffer {
	return &ringBuffer{
		packets: make([]Packet, maxPackets),
		size:    maxPackets,
	}
}

func (rb *ringBuffer) Write(p []byte) {
	if len(p) == 0 {
		return
	}

	rb.mu.Lock()
	defer rb.mu.Unlock()

	now := time.Now()
	if !rb.hasData {
		rb.startTime = now
		rb.hasData = true
	}
	rb.endTime = now

	packetData := make([]byte, len(p))
	copy(packetData, p)

	rb.packets[rb.write] = Packet{
		Timestamp: now,
		Data:      packetData,
	}

	rb.write = (rb.write + 1) % rb.size

	if rb.full {
		rb.read = (rb.read + 1) % rb.size
	}
	if rb.write == rb.read {
		rb.full = true
	}
}

func (rb *ringBuffer) ReadAvailable() ([]Packet, time.Time, time.Time) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if !rb.hasData || (!rb.full && rb.write == rb.read) {
		return nil, time.Time{}, time.Time{}
	}

	var result []Packet
	if rb.full {
		result = make([]Packet, rb.size)
		copy(result, rb.packets[rb.write:])
		copy(result[rb.size-rb.write:], rb.packets[:rb.write])
	} else if rb.write > rb.read {
		result = make([]Packet, rb.write-rb.read)
		copy(result, rb.packets[rb.read:rb.write])
	}

	start := rb.startTime
	end := rb.endTime

	rb.read = 0
	rb.write = 0
	rb.full = false
	rb.hasData = false
	rb.startTime = time.Time{}
	rb.endTime = time.Time{}

	return result, start, end
}
