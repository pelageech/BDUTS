# BDUTS | Load Balancer
<img alt="BDUTS!!! Gopher has blown up the logo" src="./logo/bduts_logo.png" width="228"><br>
The easy Go Load Balancer repository includes a distribution of requests by backends and caching.

# Getting started
## Edit configuration

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

### Load balancer
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

If you see these logs and there are alive backends, the balancer works! 

Let for each backend you have path ```/hello``` and the balancer listens on :8080. Check your load balancer:
```curl -k https://localhost:8080/hello```.

You will see the backend got the response from the balancer!
