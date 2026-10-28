package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// Изменено: убран префикс tools. перед Packet
type SampleCallback func(packets []Packet, index int, start time.Time, end time.Time)

type SampleManager struct {
	mu           sync.RWMutex
	samples      []*ringBuffer
	sampleCount  int
	currentIdx   int
	logTicker    *time.Ticker
	done         chan struct{}
	onSampleFull SampleCallback
	outputDir    string
	currentBytes int64
	maxSizeBytes int64
}

func NewSampleManager(maxPacketsPerSample, sampleCount int, callback SampleCallback) *SampleManager {
	samples := make([]*ringBuffer, sampleCount)
	for i := 0; i < sampleCount; i++ {
		samples[i] = newRingBuffer(maxPacketsPerSample)
	}

	outputDir := "out"
	_ = os.MkdirAll(outputDir, os.ModePerm)

	// 5 МБ = 5 * 1024 * 1024 байт
	maxSizeBytes := int64(5 * 1024 * 1024)

	return &SampleManager{
		samples:      samples,
		sampleCount:  sampleCount,
		currentIdx:   0,
		logTicker:    time.NewTicker(10 * time.Second),
		done:         make(chan struct{}),
		onSampleFull: callback,
		outputDir:    outputDir,
		maxSizeBytes: maxSizeBytes,
	}
}

func (sm *SampleManager) Start() {
	go func() {
		for {
			select {
			case <-sm.logTicker.C:
				sm.logCurrentProgress()
			case <-sm.done:
				sm.logTicker.Stop()
				return
			}
		}
	}()
}

func (sm *SampleManager) Stop() {
	close(sm.done)
}

func (sm *SampleManager) Write(p []byte) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	current := sm.samples[sm.currentIdx]
	current.Write(p)

	newSize := atomic.AddInt64(&sm.currentBytes, int64(len(p)))

	if newSize >= sm.maxSizeBytes {
		sm.rotateSampleWithLock()
	}
}

func (sm *SampleManager) logCurrentProgress() {
	sm.mu.RLock()
	idx := sm.currentIdx
	sm.mu.RUnlock()

	bytes := atomic.LoadInt64(&sm.currentBytes)
	kb := float64(bytes) / 1024.0

	fmt.Printf("[%s] Сэмпл №%d: записано %.2f КБ (цель: 5120.00 КБ)\n", time.Now().Format("15:04:05"), idx, kb)
}

func (sm *SampleManager) rotateSampleWithLock() {
	oldIdx := sm.currentIdx
	packets, startTime, endTime := sm.samples[oldIdx].ReadAvailable()

	if len(packets) == 0 {
		return
	}

	sm.currentIdx = (sm.currentIdx + 1) % sm.sampleCount
	atomic.StoreInt64(&sm.currentBytes, 0)

	fileName := fmt.Sprintf("sample_%d_%s.pcap", oldIdx, startTime.Format("20060102_150405"))
	filePath := filepath.Join(sm.outputDir, fileName)

	go func(path string, pks []Packet, idx int, start, end time.Time) {
		if err := SaveToPcap(path, pks); err == nil {
			var totalBytes int
			for _, p := range pks {
				totalBytes += len(p.Data)
			}
			fmt.Printf("[ЗАПИСЬ] Сэмпл %d заполнен и сохранен: %s (Размер: %.2f КБ, Пакетов: %d)\n", 
				idx, path, float64(totalBytes)/1024.0, len(pks))
		} else {
			fmt.Printf("[Ошибка] Не удалось保存 сэмпл %d: %v\n", idx, err)
		}

		if sm.onSampleFull != nil {
			sm.onSampleFull(pks, idx, start, end)
		}
	}(filePath, packets, oldIdx, startTime, endTime)
}
