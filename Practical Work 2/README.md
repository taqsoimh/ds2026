# RPC File transfer

## Usage

**Terminal 1 (or server machine)**

```bash
python rpc_server.py 9000
```

**Terminal 2 (or client machine)**

```bash
python rpc_client.py 127.0.0.1 9000 <FILE>
```

## Result

- Server prints out log with header, file name, size.

- The server.py directory will have test.bin file identical to the original.
