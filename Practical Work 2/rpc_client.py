from xmlrpc.client import ServerProxy, Binary
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

    # Create URL XML-RPC endpoint
    url = f"http://{server_host}:{port}/RPC2"
    server = ServerProxy(url, allow_none=True)

    # Read file to memory
    with open(file_path, "rb") as f:
        data = f.read()

    filename = os.path.basename(file_path)
    print(f"[+] Uploading {filename} ({len(data)} bytes) to {url}")

    # Send through RPC: Binary(...) -> binary data
    ok = server.upload_file(filename, Binary(data))

    if ok:
        print("[+] File uploaded successfully via RPC.")
    else:
        print("[-] Server reported failure.")

if __name__ == "__main__":
    main()
