from flask import Flask, jsonify, request
from flask_cors import CORS

class UploaderHTTP:
    def __init__(self, cfg, groundseg, host, port, dev):
        self.cfg = cfg
        self.gs = groundseg
        self.dev = dev
        self.host = host
        self.port = port

        self.app = Flask(__name__)
        CORS(self.app, supports_credentials=True)

        @self.app.route('/upload', methods=['POST'])
        def upload():
            if self.cfg.http_open:
                res = self.gs.handle_upload(request)
                return jsonify(res)
            else:
                return jsonify(400)

    def run(self):
        self.app.run(
                host=self.host,
                port=self.port,
                debug=self.dev,
                threaded=True,
                use_reloader=False
                )
