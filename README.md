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
  "healthCheckPeriod" : 120
}
```
where:<br>
- **"port"** which the load balancer will listen on;
- **"healthCheckPeriod"** is a period of checking if all the backends alive.

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
