class MyMiddleware:

    def __init__(self, get_response):
        self.get_response = get_response

    def __call__(self, request):
        response=self.get_response(request)
        if hasattr(request, 'path') and ('book' in request.path or 'catalog' in request.path):
            response['Cache-Control'] = "no-store"
        if hasattr(request, 'path') and 'books' in request.path:
            response['Cache-Control'] = "public"
        return response