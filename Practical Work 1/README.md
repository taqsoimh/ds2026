# Practical Work 1: TCP File transfer

## Goal: 1-1 File transfer over TCP/IP in CLI,based on the provided chat system

- One server

- One client

- Using socket

## Usage

**Terminal 1 – Server**

```bash
python server.py 9000
```

**Terminal 2 – Client**

```bash
python client.py 127.0.0.1 9000 <FILE>
```

## Result

- Server prints out log with header, file name, size.

- The server.py directory will have test.bin file identical to the original.
