package main

import (
	"io"
	"machine"
	"net"
	"strings"
	"time"

	"tinygo.org/x/drivers/netdev"
	"tinygo.org/x/drivers/netlink"
	"tinygo.org/x/drivers/rtl8720dn"
)

const (
	ssid         = "Wio-Chain-AP"
	pass         = "12345678"
	tcpListen    = ":8888"
	wifiUartBaud = 614400
)

func main() {
	machine.Serial.Configure(machine.UARTConfig{})
	waitSerial(8 * time.Second)
	println("=== Wio AP TCP <-> UART (to XIAO-C) ===")

	uart := machine.UART1
	uart.Configure(machine.UARTConfig{
		BaudRate: 9600,
		TX:       machine.UART_TX_PIN,
		RX:       machine.UART_RX_PIN,
	})
	println("[WIO] UART ready on BCM14/BCM15")

	wifi := rtl8720dn.New(&rtl8720dn.Config{
		En:       machine.RTL8720D_CHIP_PU,
		Uart:     machine.UART3,
		Tx:       machine.PB24,
		Rx:       machine.PC24,
		Baudrate: wifiUartBaud,
	})
	netdev.UseNetdev(wifi)
	link := netlink.Netlinker(wifi)
	link.NetNotify(func(event netlink.Event) {
		switch event {
		case netlink.EventNetUp:
			println("[WIO] WiFi NetUp")
		case netlink.EventNetDown:
			println("[WIO] WiFi NetDown")
		}
	})

	err := link.NetConnect(&netlink.ConnectParams{
		ConnectMode: netlink.ConnectModeAP,
		Ssid:        ssid,
		Passphrase:  pass,
		Retries:     3,
	})
	if err != nil {
		println("[WIO] NetConnect(AP) failed:", err.Error())
		for {
			time.Sleep(time.Second)
		}
	}

	println("[WIO] AP started SSID=", ssid, " TCP=", tcpListen)
	ln, err := net.Listen("tcp", tcpListen)
	if err != nil {
		println("[WIO] Listen failed:", err.Error())
		for {
			time.Sleep(time.Second)
		}
	}

	for {
		println("[WIO] waiting tcp client...")
		conn, err := ln.Accept()
		if err != nil {
			time.Sleep(200 * time.Millisecond)
			continue
		}
		println("[WIO] tcp client connected")
		handleConn(conn, uart)
	}
}

func handleConn(conn net.Conn, uart *machine.UART) {
	defer conn.Close()
	_, _ = conn.Write([]byte("WIO-AP connected. Send line text.\n"))

	tcpBuf := make([]byte, 256)
	var uartAccum [256]byte
	uartLen := 0
	lastUart := time.Now()

	flushUARTToTCP := func(force bool) bool {
		if uartLen == 0 {
			return true
		}
		if !force && time.Since(lastUart) < 20*time.Millisecond {
			return true
		}
		payload := uartAccum[:uartLen]
		_, err := conn.Write(payload)
		if err != nil {
			return false
		}
		println("[WIO][UART->TCP]", string(payload))
		uartLen = 0
		return true
	}

	for {
		_ = conn.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
		n, err := conn.Read(tcpBuf)
		if n > 0 {
			msg := string(tcpBuf[:n])
			println("[WIO][TCP->UART]", msg)
			_, _ = uart.Write(tcpBuf[:n])
		}
		if err != nil {
			if err == io.EOF {
				println("[WIO] tcp closed")
				return
			}
			if !isTimeoutErr(err) {
				println("[WIO] tcp read error:", err.Error())
				return
			}
		}

		if uart.Buffered() > 0 {
			b, _ := uart.ReadByte()
			lastUart = time.Now()
			if uartLen < len(uartAccum) {
				uartAccum[uartLen] = b
				uartLen++
			}
			if b == '\n' || uartLen == len(uartAccum) {
				if !flushUARTToTCP(true) {
					println("[WIO] tcp write error")
					return
				}
			}
		}

		if !flushUARTToTCP(false) {
			println("[WIO] tcp write error")
			return
		}
		time.Sleep(time.Millisecond)
	}
}

func waitSerial(timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for {
		if machine.Serial.DTR() || time.Now().After(deadline) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	time.Sleep(150 * time.Millisecond)
}

func isTimeoutErr(err error) bool {
	if err == nil {
		return false
	}
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true
	}
	return strings.Contains(err.Error(), "i/o timeout")
}
