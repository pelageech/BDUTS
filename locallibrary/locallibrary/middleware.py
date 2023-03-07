class MyMiddleware:

    def __init__(self, get_response):
        self.get_response = get_response

    def __call__(self, request):
        response=self.get_response(request)
        #Настройка заголовков кэша

        #для всех
        if hasattr(request, 'path') and 'catalog' in request.path:
            # т.е. мы это для такой URL подключаем: http://127.0.0.1/catalog/, главное что в ней содержится слово catalog
            response['Cache-Control'] = "no-store,  no-cache, must-revalidate"
        if hasattr(request, 'path') and 'login' in request.path:
            response['Cache-Control'] = "no-store"

        #для авторизованных пользователей
        if request.user.is_authenticated:
            if hasattr(request, 'path') and ('books' in request.path or 'book' in request.path):
                response['Cache-Control'] = "private, no-transform, max-age=600"
            if hasattr(request, 'path') and ('authors' in request.path or 'author' in request.path):
                response['Cache-Control'] = "private, max-age=600"
            if hasattr(request, 'path') and 'mybooks' in request.path:
                response['Cache-Control'] = "private, must-revalidate, max-age=600"
        
        # для неавторизованных пользователей
        if request.user.is_anonymous:
            if hasattr(request, 'path') and ('books' in request.path or 'book' in request.path):
                response['Cache-Control'] = "no-transform, public, max-age=600"
            if hasattr(request, 'path') and ('authors' in request.path or 'author' in request.path):
                response['Cache-Control'] = "public, max-age=600"

        return response