# Flask
from flask import Flask, jsonify, request, make_response
from flask_cors import CORS

# GroundSeg modules
from log import Log
from utils import Utils

# Create flask app
class GroundSeg:
    def __init__(self, config, orchestrator, ws_util):
        self.config_object = config
        self.config = config.config
        self.orchestrator = orchestrator
        self.ws_util = ws_util

        self.app = Flask(__name__)
        CORS(self.app, supports_credentials=True)

        #
        #   Routes
        #

        # temp get regions
        @self.app.route("/get-regions", methods=['GET'])
        def temp_get_regions():
            approved, message = self.verify(request)
            if approved:
                endpoint = self.config['endpointUrl']
                api_version = self.config['apiVersion']
                url = f"https://{endpoint}/{api_version}"
                if self.orchestrator.wireguard.get_regions(url):
                    return jsonify(200)
                return jsonify(400)

            return message

        # Check if cookie is valid
        @self.app.route("/cookies", methods=['GET'])
        def check_cookies():
            approved, message = self.verify(request)
            if approved:
                return jsonify(200)

            return message

        # List of Urbit Ships in Home Page
        @self.app.route("/urbits", methods=['GET'])
        def all_urbits():
            approved, message = self.verify(request)

            if approved:
                urbs = self.orchestrator.get_urbits()
                return make_response(jsonify(urbs))

            return message

        # Handle urbit ID related requests
        @self.app.route('/urbit', methods=['GET','POST'])
        def urbit_info():
            approved, message = self.verify(request)

            if approved:
                urbit_id = request.args.get('urbit_id')
                if request.method == 'GET':
                    urb = orchestrator.get_urbit(urbit_id)
                    return jsonify(urb)

                if request.method == 'POST':
                    blob = request.get_json()
                    res = self.orchestrator.urbit_post(urbit_id, blob)
                    return self.custom_jsonify(res)

            return message


        # Handle device's system settings
        @self.app.route("/system", methods=['GET','POST'])
        def system_settings():
            approved, message = self.verify(request)

            if approved:
                if request.method == 'GET':
                    return jsonify(self.orchestrator.get_system_settings())

                if request.method == 'POST':
                    module = request.args.get('module')
                    body = request.get_json()
                    sid = request.cookies.get('sessionid')
                    res = self.orchestrator.system_post(module, body, sid)
                    return jsonify(res)

            return message

        # Handle linux updates
        @self.app.route("/linux/updates", methods=['GET','POST'])
        def linux_updates():
            approved, message = self.verify(request)

            if approved:
                if request.method == 'GET':
                    return jsonify({"system_update":True})
                if request.method == 'POST':
                    res = self.orchestrator.update_restart_linux()
                    return jsonify(res)

        # Handle anchor registration related information
        @self.app.route("/anchor", methods=['GET'])
        def anchor_settings():
            approved, message = self.verify(request)

            if approved:
                res = self.orchestrator.get_anchor_settings()
                return jsonify(res)

            return message

        # Bug Reporting
        @self.app.route("/bug", methods=['POST'])
        def bug_report():
            approved, message = self.verify(request)

            if approved:
                blob = request.get_json()
                res = self.orchestrator.handle_report(blob)
                return jsonify(res)

            return message
        
        # Pier upload
        @self.app.route("/upload", methods=['POST'])
        def pier_upload():
            approved, message = self.verify(request)

            if approved:
                res = self.orchestrator.handle_upload(request)
                return jsonify(res)

            return message
        
        # Pier upload status
        @self.app.route("/upload/progress", methods=['POST'])
        def pier_upload_status():
            approved, message = self.verify(request)

            if approved:
                blob = request.get_json()
                res = self.orchestrator.upload_status(blob)
                return jsonify(res)

            return message

        # Login
        @self.app.route("/login", methods=['POST'])
        def login():
            if self.orchestrator.config['firstBoot']:
                return jsonify('setup')

            return self.orchestrator.handle_login_request(request.get_json())

        # Request for pubkey
        @self.app.route("/login/key", methods=['GET'])
        def ask_key():
            res = Utils.convert_pub(self.config_object.login_keys['cur']['pub'])
            return jsonify(res)

        # Get login status
        @self.app.route("/login/status", methods=['GET'])
        def login_status():
            res = self.orchestrator.handle_login_status()
            return jsonify(res)

        # Setup
        @self.app.route("/setup", methods=['POST'])
        def setup():
            if not self.config['firstBoot']:
                return jsonify(400)

            page = request.args.get('page')
            res = self.orchestrator.handle_setup(page, request.get_json())

            return jsonify(res)


    # Check if user is authenticated
    def verify(self, req):
        # User hasn't setup GroundSeg
        if self.config['firstBoot']:
            return (False, jsonify('setup'))

        # Session ID in url arg
        sessionid = req.args.get('sessionid')

        # Session ID as cookie
        if len(str(sessionid)) != 64:
            sessionid = req.cookies.get('sessionid')

        # Verified session
        if sessionid in self.config['sessions']:
            return (True, None)

        # No session ID provided
        return (False, jsonify(404))

    # Custom jsonify
    def custom_jsonify(self, val):
        if type(val) is int:
            return jsonify(val)
        if type(val) is str:
            return jsonify(val)
        return val

    # Run Flask app
    def run(self):
        Log.log("GroundSeg: Starting Flask server")
        debug_mode = self.config_object.debug_mode
        self.app.run(host='0.0.0.0', port=27016, threaded=True, debug=debug_mode, use_reloader=False)
