import os
from werkzeug.utils import secure_filename

class Uploader:
    def __init__(self,parent,cfg):
        self.app = parent
        self.cfg = cfg
        # TODO: check if upload being paused
        self.make_free()

    def make_free(self):
        self.cfg.http_open = False
        self.status = "free"
        self.patp = None
        self.size = 0
        self.uploaded = 0
        self.cfg.upload_secret = ''

    def update_metadata(self,patp,size):
        self.patp = patp
        self.size = size
        self.status = "uploading"

    def open_http(self,secret):
        self.cfg.http_open = True
        self.cfg.upload_secret = secret
        return self.cfg.http_open

    def close_http(self):
        self.cfg.http_open = False
        return not self.cfg.http_open

    def handle_chunk(self,req):
        # TODO disable updates

        # Get configuration
        error, remote, fix, secret, file = self.handle_configuration(req)
        if error:
            return "Invalid file type"

        filename = secure_filename(file.filename)
        patp = filename.split('.')[0]

        # Create Subfolder
        file_subfolder = f"{self.cfg.base}/uploaded/{patp}"
        os.makedirs(file_subfolder, exist_ok=True)

        save_path = f"{file_subfolder}/{filename}"
        current_chunk = int(req.form['dzchunkindex'])

        if current_chunk == 0 and os.path.exists(save_path):
            os.remove(save_path)

        try:
            with open(save_path, 'ab') as f:
                f.seek(int(req.form['dzchunkbyteoffset']))
                f.write(file.stream.read())
        except Exception as e:
            print(f"{patp}: Error writing to disk: {e}")
            self.status = "failed"
            return "Can't write to disk"

        on_disk = os.path.getsize(save_path)
        self.uploaded = on_disk
        total_chunks = int(req.form['dztotalchunkcount'])
        total_size = int(req.form['dztotalfilesize'])

        if current_chunk + 1 == total_chunks:
            # This was the last chunk, the file should be complete and the size we expect
            if self.last_chunk(patp, on_disk, total_size):
                return self.app.urbit.boot_existing(filename, remote, fix)
        else:
            # Not final chunk yet
            return 200

        return 400

    def last_chunk(self,patp, on_disk, total_size):
        if on_disk != total_size:
            print(f"uploader:last_chunk:{patp}: File size mismatched")
            self.status = "failed"
            return "File size mismatched"
        else:
            print(f"uploader:last_chunk:{patp}: Upload complete")
            print("to booooootttttttt")
            self.status = "process"
            return True

    def handle_configuration(self,req):
        error = True
        try:
            for f in req.files:
                con = f
                break
            conf = con.split('-')
            if conf[0] != 'pier':
                raise Exception("wrong_format")

            remote = conf[1] == 'remote'
            fix = conf[2] == 'yes'
            secret = conf[3]
            file = req.files[con]
            error = False

        except Exception as e:
            print(f"uploader:handle_chunk: File request fail: {e}")
            self.status = "failed"
        return error, remote, fix, secret, file
