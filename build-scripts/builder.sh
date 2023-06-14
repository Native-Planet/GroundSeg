rm -f /binary/groundseg
python3.11 -m pip install -r /api/requirements.txt
python3.11 -m nuitka --clang --onefile /api/groundseg.py -o groundseg-bin --include-package=websockets --onefile-tempdir-spec="%TEMP%/groundseg/bin"
#--include-data-dir=/data=data 
mv groundseg-bin /binary/groundseg
