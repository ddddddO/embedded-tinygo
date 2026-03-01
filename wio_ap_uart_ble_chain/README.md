# 実験: Wio AP -> UART -> XIAO Central -> BLE -> XIAO Peripheral

## 構成

スマホ(TCP client) -LAN-> Wio Terminal(WiFi AP/TCP server) -> UART -> XIAO-C(BLE Central) -> BLE(NUS) -> XIAO-P(BLE Peripheral)

- Wio: `wio_wifi_ap/main.go`
- XIAO-C: `xiao_ble_central/main.go`
- XIAO-P: `xiao_ble_peripheral/main.go`

## 役割

- Wio AP
  - SSID: `Wio-Chain-AP`
  - TCP: `:8888`
  - TCP受信をUARTへ送る
- XIAO-C
  - UART受信を BLE `UARTRX(6E400002...)` へ書き込み
  - XIAO-PからのNotifyをUARTへ返す
- XIAO-P
  - BLE受信をログ出力
  - 受信内容に `P-ACK:` を付けてNotify返送

## フラッシュ例

### Wio Terminal

```powershell
tinygo flash --target wioterminal --size short .\wio_ap_uart_ble_chain\wio_wifi_ap\
```

### XIAO-P (Peripheral)

```powershell
tinygo flash --target xiao-ble --size short .\wio_ap_uart_ble_chain\xiao_ble_peripheral\
```

### XIAO-C (Central)

```powershell
tinygo flash --target xiao-ble --size short .\wio_ap_uart_ble_chain\xiao_ble_central\
```

## ログ確認ポイント

- Wio
  - `[WIO][TCP->UART] ...`
- XIAO-C
  - `[XIAO-C][UART->BLE] ...`
- XIAO-P
  - `[XIAO-P][BLE RX] ...`

ACK戻りを確認する場合:

- XIAO-P: `[XIAO-P][BLE TX] P-ACK:...`
- XIAO-C: `[XIAO-C][BLE->UART] P-ACK:...`
- Wio: `[WIO][UART->TCP] P-ACK:...`

## スマホから送信

APに接続後、TCPクライアントで `192.168.1.80:8888` に接続し、テキストを送信してください（改行付き推奨）。
TCPクライアントには、JuiceSSHを利用(Telnet)

## memo
- このディレクトリのコードでスマホからメッセージ送信して、ペリフェラルのログに出たので、[このスレ](https://x.com/ddddddOpppppp/status/2027913478506484142?s=20) の確認ができたことになる