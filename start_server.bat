@ECHO OFF

ECHO Starting Nginx...
cd "nginx_server\"
start nginx.exe
cd "..\locallibrary\"
ECHO =====================
ECHO Starting Django...
py -3 runserver.py
PAUSE