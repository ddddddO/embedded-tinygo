- Wio Terminal <-> MH-Z19B をUART経由でCO2濃度取得・出力
  - Wio Terminal 側の5vピンとつなぐ。その際、5vピンを有効化しないといけない（コードに有効化あり）

- 以下で書き込み
    ```sh
    tinygo flash --target wioterminal --size short .\wio_to_mhz19\
    ```

- ytermを起動
    ```sh
    yterm --target wioterminal
    ```