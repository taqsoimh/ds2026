from xmlrpc.server import SimpleXMLRPCServer, SimpleXMLRPCRequestHandler
import sys
import os

class RequestHandler(SimpleXMLRPCRequestHandler):
    rpc_paths = ("/RPC2",)

def upload_file(filename, file_data):
    """
    Remote procedure:
      - filename: string
      - file_data: object have .data (bytes) by Binary(...) send
    """
    
    safe_name = os.path.basename(filename)
    data_bytes = file_data.data

    print(f"[+] Saving file {safe_name} ({len(data_bytes)} bytes)")
    with open(safe_name, "wb") as f:
        f.write(data_bytes)
    print("[+] Done.")
    return True 

def main():
    if len(sys.argv) != 2:
        print(f"Usage: {sys.argv[0]} <port>")
        sys.exit(1)

    port = int(sys.argv[1])

    # Create XML-RPC server on 0.0.0.0:port
    with SimpleXMLRPCServer(
        ("0.0.0.0", port),
        requestHandler=RequestHandler,
        allow_none=True
    ) as server:
        print(f"[+] XML-RPC server listening on 0.0.0.0:{port}")

        # Register RPC
        server.register_function(upload_file, "upload_file")

        # Loop received request RPC
        server.serve_forever()

if __name__ == "__main__":
    main()
