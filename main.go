package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"

	"shkvarka/dev/tools"
)

const (
	MaxPacketsPerSample = 200000 
	SamplesCount        = 5
)

func main() {
	// 1. Меню выбора интерфейсов Windows (\Device\NPF_...)
	deviceName := selectInterfaceMenu()
	fmt.Printf("\n[Запуск] Начинаем захват с устройства: %s\n\n", deviceName)

	// 2. Инициализация менеджера сэмплов
	analyzeCallback := func(packets []tools.Packet, index int, start time.Time, end time.Time) {
		fmt.Printf("--- [Анализ] Сэмпл №%d отправлен в обработку (пакетов: %d) ---\n", index, len(packets))
	}

	// ИСПРАВЛЕНО: передаем ровно 3 аргумента (int, int, tools.SampleCallback)
	manager := tools.NewSampleManager(MaxPacketsPerSample, SamplesCount, analyzeCallback)
	manager.Start()
	defer manager.Stop()

	// 3. Вызов OpenLive (совместимый с Windows Npcap)
	handle, err := pcap.OpenLive(deviceName, 65535, true, pcap.BlockForever)
	if err != nil {
		log.Fatalf("Не удалось открыть интерфейс: %v. Вы запустили консоль от имени Администратора?", err)
	}
	defer handle.Close()

	// 4. Потоковое чтение пакетов
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for packet := range packetSource.Packets() {
		data := packet.Data()
		if len(data) > 0 {
			manager.Write(data)
		}
	}
}

func selectInterfaceMenu() string {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		log.Fatalf("Не удалось получить список сетевых карт: %v", err)
	}

	if len(devices) == 0 {
		log.Fatal("Сетевые интерфейсы не найдены. Убедитесь, что в системе установлен Npcap.")
	}

	fmt.Println("=== Выберите сетевой интерфейс для записи трафика ===")
	for i, dev := range devices {
		var ips []string
		for _, addr := range dev.Addresses {
			ips = append(ips, addr.IP.String())
		}
		ipStr := "нет IP"
		if len(ips) > 0 {
			ipStr = strings.Join(ips, ", ")
		}
		desc := dev.Description
		if desc == "" {
			desc = "Нет описания"
		}
		fmt.Printf("[%d] %s (%s)\n", i+1, desc, ipStr)
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("\nВведите номер интерфейса (1-%d): ", len(devices))
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Ошибка ввода, попробуйте еще раз.")
			continue
		}

		input = strings.TrimSpace(input)
		num, err := strconv.Atoi(input)
		if err != nil || num < 1 || num > len(devices) {
			fmt.Printf("Неверный выбор. Введите число от 1 до %d.\n", num)
			continue
		}

		return devices[num-1].Name
	}
}