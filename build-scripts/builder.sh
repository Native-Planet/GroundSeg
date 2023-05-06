rm -f /binary/groundseg
python3.11 -m pip install -r /api/requirements.txt
python3.11 -m nuitka --clang --onefile /api/groundseg.py -o groundseg-bin --include-package=websockets --include-data-dir=/data=data --onefile-tempdir-spec="%TEMP%/groundseg/bin"
mv groundseg-bin /binary/groundseg
