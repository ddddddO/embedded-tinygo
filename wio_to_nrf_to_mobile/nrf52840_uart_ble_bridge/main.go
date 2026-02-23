package main

import (
	"machine"
	"time"

	"tinygo.org/x/bluetooth"
)

var (
	adapter = bluetooth.DefaultAdapter

	serviceUUID = bluetooth.NewUUID([16]byte{0x6E, 0x40, 0x00, 0x01, 0xB5, 0xA3, 0xF3, 0x93, 0xE0, 0xA9, 0xE5, 0x0E, 0x24, 0xDC, 0xCA, 0x9E})
	rxUUID      = bluetooth.NewUUID([16]byte{0x6E, 0x40, 0x00, 0x02, 0xB5, 0xA3, 0xF3, 0x93, 0xE0, 0xA9, 0xE5, 0x0E, 0x24, 0xDC, 0xCA, 0x9E})
	txUUID      = bluetooth.NewUUID([16]byte{0x6E, 0x40, 0x00, 0x03, 0xB5, 0xA3, 0xF3, 0x93, 0xE0, 0xA9, 0xE5, 0x0E, 0x24, 0xDC, 0xCA, 0x9E})

	txChar bluetooth.Characteristic

	bleRxBuffer [64]byte
	bleRxLen    int
	bleRxReady  bool
)

func main() {
	time.Sleep(3 * time.Second)
	println("=== nRF52840 UART-BLE Bridge ===")

	// UART設定（WIO Terminalとの通信用）
	uart := machine.DefaultUART
	uart.Configure(machine.UARTConfig{
		BaudRate: 9600,
		TX:       machine.UART_TX_PIN, // D6
		RX:       machine.UART_RX_PIN, // D7
	})
	println("UART configured")

	// BLE初期化
	must("enable BLE", adapter.Enable())

	adv := adapter.DefaultAdvertisement()
	must("config adv", adv.Configure(bluetooth.AdvertisementOptions{
		LocalName: "WIO-BLE",
	}))
	must("start adv", adv.Start())

	println("BLE ready! Device: WIO-BLE")

	must("add service", adapter.AddService(&bluetooth.Service{
		UUID: serviceUUID,
		Characteristics: []bluetooth.CharacteristicConfig{
			{
				Handle: &txChar,
				UUID:   txUUID,
				Flags:  bluetooth.CharacteristicNotifyPermission | bluetooth.CharacteristicReadPermission,
			},
			{
				UUID:  rxUUID,
				Flags: bluetooth.CharacteristicWritePermission | bluetooth.CharacteristicWriteWithoutResponsePermission,
				WriteEvent: func(client bluetooth.Connection, offset int, value []byte) {
					if !bleRxReady && len(value) < len(bleRxBuffer) {
						bleRxLen = copy(bleRxBuffer[:], value)
						bleRxReady = true
					}
				},
			},
		},
	}))

	println("\n>>> Press buttons on WIO Terminal")
	println(">>> Messages will be sent via BLE\n")

	uartBuffer := make([]byte, 64)
	lineBuffer := make([]byte, 0, 128)

	for {
		// BLE受信処理（スマホ→WIO Terminal）
		if bleRxReady {
			data := bleRxBuffer[:bleRxLen]
			bleRxReady = false

			print("[BLE->UART] ")
			println(string(data))

			// WIO TerminalにUART送信
			uart.Write(data)
			uart.Write([]byte("\r\n"))
		}

		// UART受信処理（WIO Terminal→BLE）
		if uart.Buffered() > 0 {
			n, err := uart.Read(uartBuffer)
			if err != nil {
				println("UART read error:", err.Error())
			} else if n > 0 {
				// 受信データを処理
				for i := 0; i < n; i++ {
					ch := uartBuffer[i]

					// 改行で1メッセージとして扱う
					if ch == '\n' || ch == '\r' {
						if len(lineBuffer) > 0 {
							print("[UART->BLE] ")
							println(string(lineBuffer))

							// BLE送信
							_, err := txChar.Write(lineBuffer)
							if err != nil {
								println("BLE TX error:", err.Error())
							}

							lineBuffer = lineBuffer[:0]
						}
					} else {
						lineBuffer = append(lineBuffer, ch)
					}
				}
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
