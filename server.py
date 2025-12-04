import socket
import struct
import sys
import os

# Read n bytes from socket
def recv_exact(sock, n):
    data = b""
    while len(data) < n:
        chunk = sock.recv(n - len(data))
        if not chunk:
            raise ConnectionError("Connection closed while receiving data")
        data += chunk
    return data

def main():
    if len(sys.argv) != 2:
        print(f"Usage: {sys.argv[0]} <port>")
        sys.exit(1)

    port = int(sys.argv[1])

    # 1. Create TCP socket = socket()
    listen_sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)

    # 2. setsockopt(SO_REUSEADDR) -> bind port 
    listen_sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)

    # 3. bind() to 0.0.0.0:port
    listen_sock.bind(("0.0.0.0", port))

    # 4. listen() wait client connection
    listen_sock.listen(1)
    print(f"[+] Listening on 0.0.0.0:{port} ...")

    # 5. accept() 1 connection
    conn, addr = listen_sock.accept()
    print(f"[+] Connection from {addr}")

    # === BEGIN PROTOCOL ===
    header_fmt = "!BHQ" # B=1 byte, H=2 bytes, Q=8 bytes, big-endian
    header_size = struct.calcsize(header_fmt)

    # 6. recv() header
    header_data = recv_exact(conn, header_size)
    opcode, name_len, file_size = struct.unpack(header_fmt, header_data)

    if opcode != 0x01:
        print("[-] Unsupported opcode:", opcode)
        conn.close()
        listen_sock.close()
        return

    print(f"[+] Header received: name_len={name_len}, file_size={file_size}")

    # 7. recv() file's name
    filename_bytes = recv_exact(conn, name_len)
    filename = filename_bytes.decode("utf-8")

    # (optional) not able to overwrite file
    if os.path.exists(filename):
        print(f"[!] File {filename} exists, will overwrite.")

    print(f"[+] Receiving file: {filename}")

    # 8. recv() data and write to file
    remaining = file_size
    with open(filename, "wb") as f:
        while remaining > 0:
            chunk_size = 4096 if remaining >= 4096 else remaining
            chunk = conn.recv(chunk_size)
            if not chunk:
                raise ConnectionError("Connection closed while receiving file data")
            f.write(chunk)
            remaining -= len(chunk)

    print(f"[+] Done. Saved to {filename}")

    # 9. Close socket
    conn.close()
    listen_sock.close()

if __name__ == "__main__":
    main()
