package main

import (
	"machine"
	"strings"
	"time"

	"tinygo.org/x/bluetooth"
)

var adapter = bluetooth.DefaultAdapter

const targetName = "RELAY-P"

var (
	bleRxBuf   [128]byte
	bleRxLen   int
	bleRxReady bool
)

func main() {
	time.Sleep(2 * time.Second)
	println("=== XIAO-C UART <-> BLE Central ===")

	uart := machine.DefaultUART
	uart.Configure(machine.UARTConfig{BaudRate: 9600, TX: machine.UART_TX_PIN, RX: machine.UART_RX_PIN})
	println("[XIAO-C] UART ready D6/D7")

	must("enable BLE", adapter.Enable())

	for {
		device, rxChar, err := connectAndPrepare()
		if err != nil {
			println("[XIAO-C] connect flow error:", err.Error())
			time.Sleep(2 * time.Second)
			continue
		}
		println("[XIAO-C] connected and ready")

		uartBuf := make([]byte, 64)
		lineBuf := make([]byte, 0, 160)

		alive := true
		for alive {
			if bleRxReady {
				msg := bleRxBuf[:bleRxLen]
				bleRxReady = false
				print("[XIAO-C][BLE->UART] ")
				println(string(msg))
				_, _ = uart.Write(msg)
				_, _ = uart.Write([]byte("\n"))
			}

			if uart.Buffered() > 0 {
				n, err := uart.Read(uartBuf)
				if err != nil {
					println("[XIAO-C] uart read error:", err.Error())
				} else if n > 0 {
					for i := 0; i < n; i++ {
						ch := uartBuf[i]
						if ch == '\n' || ch == '\r' {
							if len(lineBuf) > 0 {
								print("[XIAO-C][UART->BLE] ")
								println(string(lineBuf))
								if _, err := rxChar.WriteWithoutResponse(lineBuf); err != nil {
									println("[XIAO-C] BLE write error:", err.Error())
									alive = false
									break
								}
								lineBuf = lineBuf[:0]
							}
						} else {
							lineBuf = append(lineBuf, ch)
						}
					}
				}
			}

			time.Sleep(10 * time.Millisecond)
		}

		_ = device.Disconnect()
		println("[XIAO-C] disconnected; retry")
		time.Sleep(1200 * time.Millisecond)
	}
}

func connectAndPrepare() (bluetooth.Device, bluetooth.DeviceCharacteristic, error) {
	addr, err := scanTarget(targetName, 8*time.Second)
	if err != nil {
		return bluetooth.Device{}, bluetooth.DeviceCharacteristic{}, err
	}

	device, err := adapter.Connect(addr, bluetooth.ConnectionParams{})
	if err != nil {
		return bluetooth.Device{}, bluetooth.DeviceCharacteristic{}, err
	}

	services, err := device.DiscoverServices([]bluetooth.UUID{bluetooth.ServiceUUIDNordicUART})
	if err != nil || len(services) == 0 {
		_ = device.Disconnect()
		if err != nil {
			return bluetooth.Device{}, bluetooth.DeviceCharacteristic{}, err
		}
		return bluetooth.Device{}, bluetooth.DeviceCharacteristic{}, errString("NUS service missing")
	}

	rxChars, err := services[0].DiscoverCharacteristics([]bluetooth.UUID{bluetooth.CharacteristicUUIDUARTRX})
	if err != nil || len(rxChars) == 0 {
		_ = device.Disconnect()
		if err != nil {
			return bluetooth.Device{}, bluetooth.DeviceCharacteristic{}, err
		}
		return bluetooth.Device{}, bluetooth.DeviceCharacteristic{}, errString("NUS RX missing")
	}

	txChars, err := services[0].DiscoverCharacteristics([]bluetooth.UUID{bluetooth.CharacteristicUUIDUARTTX})
	if err != nil || len(txChars) == 0 {
		_ = device.Disconnect()
		if err != nil {
			return bluetooth.Device{}, bluetooth.DeviceCharacteristic{}, err
		}
		return bluetooth.Device{}, bluetooth.DeviceCharacteristic{}, errString("NUS TX missing")
	}

	bleRxReady = false
	err = txChars[0].EnableNotifications(func(data []byte) {
		if bleRxReady || len(data) == 0 || len(data) > len(bleRxBuf) {
			return
		}
		bleRxLen = copy(bleRxBuf[:], data)
		bleRxReady = true
	})
	if err != nil {
		_ = device.Disconnect()
		return bluetooth.Device{}, bluetooth.DeviceCharacteristic{}, err
	}

	println("[XIAO-C] notify enabled")
	return device, rxChars[0], nil
}

func scanTarget(name string, timeout time.Duration) (bluetooth.Address, error) {
	println("[XIAO-C] scanning:", name)
	found := false
	var foundAddr bluetooth.Address
	start := time.Now()

	err := adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
		if time.Since(start) > timeout {
			_ = adapter.StopScan()
			return
		}
		if !found && result.LocalName() == name {
			found = true
			foundAddr = result.Address
			println("[XIAO-C] scan hit:", result.Address.String())
			_ = adapter.StopScan()
		}
	})
	if err != nil {
		if !(found && isInvalidStateErr(err)) {
			return bluetooth.Address{}, err
		}
	}
	if !found {
		return bluetooth.Address{}, errString("target not found")
	}
	return foundAddr, nil
}

type errString string

func (e errString) Error() string { return string(e) }

func must(action string, err error) {
	if err != nil {
		panic("failed to " + action + ": " + err.Error())
	}
}

func isInvalidStateErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "invalid state") || strings.Contains(msg, "operation disallowed")
}
