class UploadBroadcast:
    def __init__(self, groundseg):
        self.app = groundseg
        self.uploader = self.app.uploader

    def display(self):
        return {
                "status":self.uploader.status,
                "size":self.uploader.size,
                "uploaded":self.uploader.uploaded,
                "patp":self.uploader.patp,
                }
