package main

import (
	"fmt"
	"machine"
	"time"
)

func main() {
	time.Sleep(5 * time.Second)
	fmt.Println("start")

	power5V := machine.OUTPUT_CTR_5V
	power5V.Configure(machine.PinConfig{Mode: machine.PinOutput})
	power5V.High()
	fmt.Println("5V power enabled")

	// 電源安定化待ち
	time.Sleep(2 * time.Second)

	// Wio Terminalの背面ピンを直接指定
	tx := machine.UART_TX_PIN // UART1 TX (Pin 8)
	rx := machine.UART_RX_PIN // UART1 RX (Pin 10)

	uart := machine.UART1
	err := uart.Configure(machine.UARTConfig{
		BaudRate: 9600,
		TX:       tx,
		RX:       rx,
	})
	if err != nil {
		fmt.Println("UART Configuration Error:", err)
		return
	}
	fmt.Println("UART configured")

	// 読み取りコマンド
	cmd := []byte{0xFF, 0x01, 0x86, 0x00, 0x00, 0x00, 0x00, 0x00, 0x79}
	buf := make([]byte, 9)

	println("MH-Z19B Warm-up... (Waiting 5s)")
	time.Sleep(5 * time.Second)

	for {
		uart.Write(cmd)

		time.Sleep(100 * time.Millisecond)

		if uart.Buffered() >= 9 {
			n, _ := uart.Read(buf)
			if n == 9 && buf[0] == 0xFF && buf[1] == 0x86 {
				co2 := int(buf[2])*256 + int(buf[3])
				println("CO2 Concentration:", co2, "ppm")
			}
		}

		time.Sleep(2 * time.Second)
	}
}
