rm -f /binary/groundseg
python3.10 -m pip install -r /api/requirements.txt
python3.10 -m nuitka --clang --onefile /api/groundseg.py -o groundseg-bin
mv groundseg-bin /binary/groundseg
