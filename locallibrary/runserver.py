#runserver.py
from waitress import serve

from locallibrary.wsgi import application
print("Starting server...")
if __name__ == '__main__':
    serve(application, host='127.0.0.1', port='8080')