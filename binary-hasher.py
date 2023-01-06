import hashlib

file = "binary/groundseg"
h  = hashlib.sha256()
b  = bytearray(128*1024)
mv = memoryview(b)
with open(file, 'rb', buffering=0) as f:
    while n := f.readinto(mv):
        h.update(mv[:n])
print(h.hexdigest())
