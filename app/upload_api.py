import requests, copy, json, shutil
from flask import Flask, flash, request, redirect, url_for, send_from_directory, Response, Blueprint
from flask import render_template, make_response, jsonify
from flask import current_app

import os
import zipfile, tarfile
import glob
from werkzeug.utils import secure_filename

from orchestrator import Orchestrator

import urbit_docker

#import system_info as sys_info


app = Blueprint('upload', __name__, template_folder='templates')


def make_urbit(patp, http_port, ames_port):
    data = copy.deepcopy(urbit_docker.default_pier_config)
    data['pier_name'] = patp
    data['http_port'] = http_port
    data['ames_port'] = ames_port
    with open(f'settings/pier/{patp}.json', 'w') as f:
        json.dump(data, f, indent = 4)
    
    urbit = urbit_docker.UrbitDocker(data)
    return urbit


@app.route('/upload/key',methods=['POST'])
def uploadKey():
    if request.method == 'POST':
        patp = request.form['patp']
        key = request.form['key']

        http_port, ames_port = current_app.config['ORCHESTRATOR'].getOpenUrbitPort()
        urbit = make_urbit(patp, http_port, ames_port)
        urbit.addKey(key)
        current_app.config['ORCHESTRATOR'].addUrbit(patp, urbit)

        return jsonify(200)
        

@app.route('/upload/pier', methods=['POST'])
def uploadPier():
    if request.method == 'POST':

        file = request.files['file']

        filename = secure_filename(file.filename)
        fn = save_path = os.path.join(current_app.config['TEMP_FOLDER'],filename)
        current_chunk = int(request.form['dzchunkindex'])
        
        if os.path.exists(save_path) and current_chunk == 0:
            # 400 and 500s will tell dropzone that an error occurred and show an error
            os.remove(os.path.join(current_app.config['TEMP_FOLDER'], filename))
            return make_response(('File already exists', 400))

        try:
            with open(save_path, 'ab') as f:
                f.seek(int(request.form['dzchunkbyteoffset']))
                f.write(file.stream.read())
        except OSError:
            # log.exception will include the traceback so we can see what's wrong
            print('Could not write to file')
            return make_response(("Not sure why,"
                                  " but we couldn't write the file to disk", 500))


        total_chunks = int(request.form['dztotalchunkcount'])

        if current_chunk + 1 == total_chunks:
            # This was the last chunk, the file should be complete and the size we expect
            if os.path.getsize(save_path) != int(request.form['dztotalfilesize']):
                print(f"File {file.filename} was completed, "
                          f"but has a size mismatch."
                          f"Was {os.path.getsize(save_path)} but we"
                          f" expected {request.form['dztotalfilesize']} ")
                return make_response(('Size mismatch', 500))
            else:
                print(f'File {file.filename} has been uploaded successfully')
                if filename.endswith("zip"):
                    with zipfile.ZipFile(fn) as zip_ref:
                        zip_ref.extractall(current_app.config['TEMP_FOLDER']);
                elif filename.endswith("tar.gz") or filename.endswith("tgz"):
                    tar = tarfile.open(fn,"r:gz")
                    tar.extractall(app.config['TEMP_FOLDER'])
                    tar.close()

                print("extracted")

                os.remove(os.path.join(current_app.config['TEMP_FOLDER'], filename))
                timeout = 10000
                
                patp = filename[:-4]
                http_port, ames_port = current_app.config['ORCHESTRATOR'].getOpenUrbitPort()
                urbit = make_urbit(patp, http_port, ames_port)
                urbit.copyFolder(current_app.config['TEMP_FOLDER'])
                shutil.rmtree(os.path.join(current_app.config['TEMP_FOLDER'], patp))
                current_app.config['ORCHESTRATOR'].addUrbit(patp, urbit)

                print('wtf')
                return jsonify(200)
        else:
            print(f'Chunk {current_chunk + 1} of {total_chunks} '
                      f'for file {file.filename} complete')
        return make_response(("Chunk upload successful", 200))

