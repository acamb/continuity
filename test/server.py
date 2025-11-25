#!/usr/bin/env python3
from http.server import SimpleHTTPRequestHandler, HTTPServer
from functools import partial

class CookieHTTPRequestHandler(SimpleHTTPRequestHandler):
    def __init__(self, *args, cookie_name=None, cookie_value=None, **kwargs):
        self.cookie_name = cookie_name
        self.cookie_value = cookie_value
        super().__init__(*args, **kwargs)
    
    def end_headers(self):
        if self.cookie_name and self.cookie_value:
            self.send_header('Set-Cookie', f'{self.cookie_name}={self.cookie_value}; Path=/; Max-Age=3600')
        super().end_headers()

def run_server(port=8000, cookie_name='MY_CUSTOM_COOKIE', cookie_value='1'):
    handler = partial(CookieHTTPRequestHandler, 
                     cookie_name=cookie_name, 
                     cookie_value=cookie_value)
    
    server = HTTPServer(('', port), handler)
    print(f'Server started on http://localhost:{port}')
    print(f'Cookie: {cookie_name}={cookie_value}')
    
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print('\nServer stopped')

if __name__ == '__main__':
    import argparse
    
    parser = argparse.ArgumentParser(description='HTTP Server with custom cookie')
    parser.add_argument('-p', '--port', type=int, default=8080,
                       help='Server port (default: 8080)')
    parser.add_argument('-n', '--name', type=str, default='MY_CUSTOM_COOKIE',
                       help='Cookie name (default: MY_CUSTOM_COOKIE)')
    parser.add_argument('-v', '--value', type=str, default='1',
                       help='Cookie value (default: 1)')
    
    args = parser.parse_args()
    
    run_server(port=args.port, cookie_name=args.name, cookie_value=args.value)
