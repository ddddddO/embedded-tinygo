package main

import (
	"fmt"
	"machine"
	"time"
)

func main() {
	time.Sleep(3 * time.Second)
	fmt.Println("=== WIO Terminal Button to UART ===")

	// 5方向スイッチの設定
	btnUp := machine.WIO_5S_UP
	btnDown := machine.WIO_5S_DOWN
	btnLeft := machine.WIO_5S_LEFT
	btnRight := machine.WIO_5S_RIGHT
	btnPress := machine.WIO_5S_PRESS

	btnUp.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	btnDown.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	btnLeft.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	btnRight.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	btnPress.Configure(machine.PinConfig{Mode: machine.PinInputPullup})

	// UART1設定（nRF52840との通信用）
	uart := machine.UART1
	err := uart.Configure(machine.UARTConfig{
		BaudRate: 9600,
	})
	if err != nil {
		fmt.Printf("UART error: %v\r\n", err)
		return
	}

	fmt.Println("UART configured (BCM14/BCM15)")
	fmt.Println("\nPress 5-way switch to send messages:")
	fmt.Println("  UP    = Send 'Hello'")
	fmt.Println("  DOWN  = Send 'World'")
	fmt.Println("  LEFT  = Send 'Left'")
	fmt.Println("  RIGHT = Send 'Right'")
	fmt.Println("  PRESS = Send 'OK'")
	fmt.Println("\nWaiting for button press...\n")

	// ボタン状態記録（チャタリング防止）
	lastPress := time.Now()
	recvBuffer := make([]byte, 64)

	for {
		now := time.Now()

		// 100ms以上経過していればボタンチェック
		if now.Sub(lastPress) > 100*time.Millisecond {
			var message string

			// ボタンはプルアップなのでLowで押下検出
			if !btnUp.Get() {
				message = "Hello"
			} else if !btnDown.Get() {
				message = "World"
			} else if !btnLeft.Get() {
				message = "Left"
			} else if !btnRight.Get() {
				message = "Right"
			} else if !btnPress.Get() {
				message = "OK"
			}

			if message != "" {
				// UART送信
				msg := message + "\n"
				n, err := uart.Write([]byte(msg))
				if err != nil {
					fmt.Printf("Send error: %v\r\n", err)
				} else {
					fmt.Printf("[WIO->nRF] %s (%d bytes)\r\n", message, n)
				}
				lastPress = now
			}
		}

		// nRF52840からの受信チェック（BLEからの返信）
		if uart.Buffered() > 0 {
			n, err := uart.Read(recvBuffer)
			if err != nil {
				fmt.Printf("Receive error: %v\r\n", err)
			} else if n > 0 {
				fmt.Printf("[nRF->WIO] %s", string(recvBuffer[:n]))
			}
		}

		time.Sleep(10 * time.Millisecond)
	}
}
