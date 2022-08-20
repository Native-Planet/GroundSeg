from flask import Flask, flash, request, redirect, url_for, send_from_directory, Response, Blueprint
from flask import render_template, make_response
import requests

#import system_info as sys_info

app = Blueprint('urbit', __name__, template_folder='templates')

@app.route('/upload/pier', methods=['GET','POST'])
def uploadPier():
    return render_template('upload_pier.html')

@app.route('/launch/upload_pier', methods=['GET','POST'])
def upload_pier():
    if request.method == 'POST':

        file = request.files['file']

        filename = secure_filename(file.filename)
        fn = save_path = os.path.join(app.config['UPLOAD_PIER'],filename)
        current_chunk = int(request.form['dzchunkindex'])
        
        if os.path.exists(save_path) and current_chunk == 0:
            # 400 and 500s will tell dropzone that an error occurred and show an error
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
                        zip_ref.extractall(app.config['UPLOAD_PIER']);
                elif filename.endswith("tar.gz") or filename.endswith("tgz"):
                    tar = tarfile.open(fn,"r:gz")
                    tar.extractall(app.config['UPLOAD_PIER'])
                    print("extracted")
                    tar.close()


                os.remove(os.path.join(app.config['UPLOAD_PIER'], filename))
                timeout = 10000
                return redirect("/")
        else:
            print(f'Chunk {current_chunk + 1} of {total_chunks} '
                      f'for file {file.filename} complete')

        return make_response(("Chunk upload successful", 200))
    else:
        return redirect("/")

