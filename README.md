# BDUTS | Load Balancer
<img alt="BDUTS!!! Gopher has blown up the logo" src="./logo/bduts_logo.png" width="228"><br>
The easy Go Load Balancer repository includes a distribution of requests by backends and caching.

# Configuration

### Backends
Before starting you should configure your backends and put their information to ```resources/servers.json```.
For each backend use the following pattern:
```
[
    {
      "url": "http://192.168.0.1:8080",
      "healthCheckTcpTimeout": 1000
      "maximalRequests": 5
    },
    ...
]
```
where:<br>
- **"url"** is the address of the backend to which requests are sent relative to the URL of the load balancer;
- **"healthCheckTcpTimeout"** is maximum response time from the backend for a tcp packet of the health checker;
- **"maximalRequests"** is how many requests can be processed on the backend at the same time.

### Load Balancer
BDUTS uses **HTTPS** method, that's why you need to put files ```MyCertificate.crt``` and ```MyKey.key``` to the root of project.

Then, configure your load balancer in ```resources/config.json```
```
{
  "port" : 8080,
  "healthCheckPeriod" : 120,
  "maxCacheSize" : 10000000,
  "observeFrequency" : 10000
}
```
where:<br>
- **"port"** which the load balancer will listen on;
- **"healthCheckPeriod"** is a period of checking if all the backends alive;
- **"maxCacheSize"** is a maximal size _in bytes_ for storing cached pages;
- **"observeFrequency"** is a period of observing cache _in milliseconds_ and detecting whether it is necessary to delete rotten or little-used pagesю

### Caching
The last step is optional but necessary if you want caching to work properly.

For each possible path you should put their key constructors explicitly to the file ```resources/cache_config.json```.
Follow the pattern:
```
[
  {
    "location" : "/",
    "requestKey" : "REQ_METHOD;REQ_HOST;REQ_URI"
  },
  {
    "location" : "/draw",
    "requestKey" : "REQ_METHOD;REQ_HOST;REQ_URI;REQ_QUERY"
  },
  ...
]
```
Hash of the requests will be calculated based on ```requestKey```.
All the possible directives:
- **REQ_METHOD**  : include method
- **REQ_HOST**    : include host
- **REQ_URI**     : include path
- **REQ_QUERY**   : include GET-Query (after ?)

_Note: wrong directives will be ignored. If there is no any correct directive, then ```"REQ_METHOD;REQ_HOST;REQ_URI"``` is used as default._

## Let's start our balancer
Just build the project ```go build .``` and run it.
Or run immediately ```go run .```.

You will see some logs in your terminal:
- All the backends read from JSON-config;
- A result of the first health checking;
- "Load Balancer started at :port".

If you see these logs and there are alive backends, the balancer works! <br>
Let for each backend you have path ```/hello``` and the balancer listens on :8080.

Check your load balancer:
```curl -k https://localhost:8080/hello```.<br>
You will see in the logs the backend got the response from the balancer!

# Pool of Backends
When you start the load balancer, it reads all the information about backends and creates a server pool.
The server pool contains a list of backends and some data about each of them: all the fields from JSON-config and
their life status (alive or not).

The balancer uses Weighted Round-Robin algorithm (WRR) for balancing between backends. The pool has a pointer (_int index_)
looking at the last backend that the request was sent to.
The beginning value of the pointer is -1, after the first request its value is always between **0** and **len(pool) - 1**.

The algorithm chooses only alive servers. If the backend is chosen, the request will be sent to this one.<br>
WRR begins again if backend is full of requests (recall that backends have limits on the number of requests processing at the same time).

If the backend doesn't answer or it returns 5xx, it marks *not-alive*.
The backend can become alive again if it passes the next health checker test.

# Cache-Proxy
Before sending request the load balancer checks the page in cache. If there is one, the page is read from disk and returned to the client.

The metadata of the page is stored in memory, exactly in boltDB. Before reading it's checked a presence of metadata of the page we're looking for.
If everything is OK, the page is read from a disk. If there's no any errors, the page is returned to the client.
The balancer sends a request to the backend in case any error is occured.

The load balancer creates a key by key directives from ```resources/cache_config.json``` and take a hash of it with hex-encoding. The value saves into request's context.
The value always uses if it deals with cache:

_Let we have a hash with 128 length:_<br>
- Writing: the page will be saved into a specified directory and named<br>```:root:/cache-data/db/hash[0:31]/hash[32:63]/hash[64:95]/hash[96:127]/hash[:]```;
- Reading: the balancer will use this hash for searching the page on a disk by the directory above.

<hr>

#### Logo by <a href="https://kazachokolate.tumblr.com/">Kazachokolate</a>
