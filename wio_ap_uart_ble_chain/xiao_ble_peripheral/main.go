package main

import (
	"machine"
	"time"

	"tinygo.org/x/bluetooth"
)

var (
	adapter = bluetooth.DefaultAdapter
	txChar  bluetooth.Characteristic

	bleBuf   [128]byte
	bleLen   int
	bleReady bool
)

func main() {
	time.Sleep(2 * time.Second)
	println("=== XIAO-P BLE Peripheral Logger ===")

	uart := machine.DefaultUART
	uart.Configure(machine.UARTConfig{BaudRate: 9600, TX: machine.UART_TX_PIN, RX: machine.UART_RX_PIN})
	println("[XIAO-P] UART ready D6/D7")

	must("enable BLE", adapter.Enable())
	must("add service", adapter.AddService(&bluetooth.Service{
		UUID: bluetooth.ServiceUUIDNordicUART,
		Characteristics: []bluetooth.CharacteristicConfig{
			{
				Handle: &txChar,
				UUID:   bluetooth.CharacteristicUUIDUARTTX,
				Flags:  bluetooth.CharacteristicNotifyPermission | bluetooth.CharacteristicReadPermission,
			},
			{
				UUID:  bluetooth.CharacteristicUUIDUARTRX,
				Flags: bluetooth.CharacteristicWritePermission | bluetooth.CharacteristicWriteWithoutResponsePermission,
				WriteEvent: func(client bluetooth.Connection, offset int, value []byte) {
					if !bleReady && len(value) <= len(bleBuf) {
						bleLen = copy(bleBuf[:], value)
						bleReady = true
					}
				},
			},
		},
	}))

	adv := adapter.DefaultAdvertisement()
	must("config adv", adv.Configure(bluetooth.AdvertisementOptions{
		LocalName:    "RELAY-P",
		ServiceUUIDs: []bluetooth.UUID{bluetooth.ServiceUUIDNordicUART},
	}))
	must("start adv", adv.Start())
	println("[XIAO-P] advertising as RELAY-P")

	for {
		if bleReady {
			data := bleBuf[:bleLen]
			bleReady = false
			print("[XIAO-P][BLE RX] ")
			println(string(data))

			_, _ = uart.Write(data)
			_, _ = uart.Write([]byte("\n"))

			ack := append([]byte("P-ACK:"), data...)
			_, err := txChar.Write(ack)
			if err != nil {
				println("[XIAO-P] notify error:", err.Error())
			} else {
				print("[XIAO-P][BLE TX] ")
				println(string(ack))
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func must(action string, err error) {
	if err != nil {
		panic("failed to " + action + ": " + err.Error())
	}
}
