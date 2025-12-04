#!/usr/bin/env python3
import socket
import struct
import sys
import os

def main():
    if len(sys.argv) != 4:
        print(f"Usage: {sys.argv[0]} <server_host> <port> <file_path>")
        sys.exit(1)

    server_host = sys.argv[1]
    port        = int(sys.argv[2])
    file_path   = sys.argv[3]

    if not os.path.isfile(file_path):
        print("[-] File not found:", file_path)
        sys.exit(1)

    # 1. Take IP's server = gethostbyname()
    server_ip = socket.gethostbyname(server_host)

    # 2. Create TCP socket
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)

    # 3. connect() to server
    sock.connect((server_ip, port))
    print(f"[+] Connected to {server_ip}:{port}")

    # 4. Prepare metadata file
    file_size = os.path.getsize(file_path)
    filename = os.path.basename(file_path).encode("utf-8")
    name_len = len(filename)
    opcode   = 0x01

    # 5. Pack header = opcode + name_len + file_size
    header_fmt = "!BHQ"
    header = struct.pack(header_fmt, opcode, name_len, file_size)

    # 6. send() header + file's name
    sock.sendall(header)
    sock.sendall(filename)

    print(f"[+] Sending file: {file_path} ({file_size} bytes)")

    # 7. send() data
    with open(file_path, "rb") as f:
        while True:
            chunk = f.read(4096)
            if not chunk:
                break
            sock.sendall(chunk)

    print("[+] File sent successfully.")

    # 8. close()
    sock.close()

if __name__ == "__main__":
    main()
