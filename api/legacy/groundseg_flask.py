from time import sleep

# Flask
from flask import Flask, jsonify, request, make_response
from flask_cors import CORS

# GroundSeg modules
from log import Log
from config.utils import Utils

# Create flask app
class GroundSegFlask:
    def __init__(self,state):#, config, orchestrator, ws_util):
        self.state = state
        self.config_object = self.state['config']
        while self.config_object == None:
            sleep(0.5)
            self.config_object = self.state['config']
        self.config = self.config_object.config
        self.orchestrator = self.state['orchestrator']

        self.app = Flask(__name__)
        CORS(self.app, supports_credentials=True)

        #
        #   Routes
        #

        # Check if cookie is valid
        @self.app.route("/cookies", methods=['GET'])
        def check_cookies():
            approved, message = self.verify(request)
            if approved:
                return jsonify(200)

            return message

        # List of Urbit Ships in Home Page
        '''
        @self.app.route("/urbits", methods=['GET'])
        def all_urbits():
            approved, message = self.verify(request)

            if approved:
                urbs = self.orchestrator.get_urbits()
                return make_response(jsonify(urbs))

            return message
        '''

        # Handle urbit ID related requests
        @self.app.route('/urbit', methods=['GET','POST'])
        def urbit_info():
            approved, message = self.verify(request)

            if approved:
                urbit_id = request.args.get('urbit_id')
                if request.method == 'GET':
                    urb = self.orchestrator.get_urbit(urbit_id)
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
        if self.orchestrator:
            return True, None
        else:
            self.orchestrator = self.state['orchestrator']
            return False, jsonify("NOT_READY")
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
