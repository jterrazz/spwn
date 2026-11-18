# A cluster's readyz on 8443, behind the certificate it made itself.
import http.server
import ssl


class Ready(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200 if self.path == "/readyz" else 404)
        self.end_headers()
        self.wfile.write(b"ok\n")

    def log_message(self, *args):
        pass


server = http.server.HTTPServer(("0.0.0.0", 8443), Ready)
context = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
context.load_cert_chain("/tmp/e2e-tls/cert.pem", "/tmp/e2e-tls/key.pem")
server.socket = context.wrap_socket(server.socket, server_side=True)
server.serve_forever()
