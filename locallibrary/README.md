Установка

Установить Python 3
Установить виртуальную среду для джанго
pip3 install virtualenvwrapper-win
Создать виртуальную среду, запустить и установить джанго (в cmd; гайд, если нужно)
mkvirtualenv my_django_environment
workon
pip3 install django~=4.0
Открой папку проекта
Далее в папке locallibrary (в cmd):
pip3 install -r requirements.txt
py -3 manage.py makemigrations
py -3 manage.py migrate
py -3 manage.py collectstatic
Установить Waitress(связывает Nginx и Django):
pip install waitress
В файле nginx.conf настрой свой путь в 11 строке
C:/<твой путь>/BDUTS/nginx_server/sites-enabled/locallibrary.conf
Теперь открой два терминала в папках
/BDUTS/nginx_server/
/BDUTS/locallibrary/
Запускай сервер так
на первой терминале: py -3 runserver.py
на втором: start nginx
Сервер находится на 127.0.0.1:80


Структура сервера


Настройка всех хедеров находится в файле BDUTS/locallibrary/locallibrary/middleware.py, там вроде все понятно, если что добавил комменты. Там для каждой страницы на сервере правила кэширования лежат.
