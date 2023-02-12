Как запустить сервер:
1. Перейти в директорию nginx_server
2. Открыть окно PowerShell здесь
3. Запускаем Nginx командой “start nginx”
4. Проверяем запущенные процессы командой “tasklist /fi "imagename eq nginx.exe"”. Должно быть 2 процесса nginx.exe.
5. Выключение сервера “.\nginx -s quit”
6. Перезагрузка сервера “.\nginx -s reload”


conf/nginx.conf – файл настройки сервера
