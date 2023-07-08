class Config:
    def __init__(self, base, dev):
        super().__init__()
        self.base = base
        self.dev = dev
        self.internet = False

    async def netcheck(self):
        #temp
        print("temp: check internet here")
        self.internet = True
