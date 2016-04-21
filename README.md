[![Build Status](https://travis-ci.org/buger/gor.png?branch=master)](https://travis-ci.org/buger/gor) [![codebeat](https://codebeat.co/badges/6427d589-a78e-416c-a546-d299b4089893)](https://codebeat.co/projects/github-com-buger-gor) [![Go Report Card](https://goreportcard.com/badge/github.com/buger/gor)](https://goreportcard.com/report/github.com/buger/gor)

## About

Gor is an open-source tool for capturing and replaying live HTTP traffic into a test environment in order to continuously test your system with real data. It can be used to increase confidence in code deployments, configuration changes and infrastructure changes.


Now you can test your code on real user sessions in an automated and repeatable fashion.
**No more falling down in production!**

Here is basic workflow: The listener server catches http traffic and sends it to the replay server or saves to file. The replay server forwards traffic to a given address.


![Diagram](http://i.imgur.com/9mqj2SK.png)


## Examples

### Capture traffic from port
```bash
# Run on servers where you want to catch traffic. You can run it on each `web` machine.
sudo gor --input-raw :80 --output-tcp replay.local:28020

# Replay server (replay.local).
gor --input-tcp replay.local:28020 --output-http http://staging.com
```

Since Gor use raw sockets to capture traffic it require `sudo` access. Alternatively you can allow access to raw sockets like this: `sudo setcap CAP_NET_RAW=ep gor`

### Using 1 Gor instance for both listening and replaying
It's recommended to use separate server for replaying traffic, but if you have enough CPU resources you can use single Gor instance.

```
sudo gor --input-raw :80 --output-http "http://staging.com"
```

### Guarantee of replay and HTTP input
Due to how traffic interception works, there is chance of missing requests. If you want guarantee that requests will be replayed you can use http input, but it will require changes in your app as well. 

```
sudo gor --input-http :28019 --output-http "http://staging.com"
```

Then in your application you should send copy (e.g. like reverse proxy) all incoming requests to Gor http input. 

## Configuration

### Forward to multiple addresses

You can forward traffic to multiple endpoints. Just add multiple --output-* arguments.
```
gor --input-tcp :28020 --output-http "http://staging.com"  --output-http "http://dev.com"
```

#### Splitting traffic
By default it will send same traffic to all outputs, but you have options to equally split it:

```
gor --input-tcp :28020 --output-http "http://staging.com"  --output-http "http://dev.com" --split-output true
```

### HTTP output workers
By default Gor creates a dynamic pool of workers: it starts with 10 and creates more http output workers when the http output queue length is greater than 10.  The number of workers created (N) is equal to the queue length at the time which it is checked and found to have a length greater than 10. The queue length is checked every time a message is written to the http output queue.  No more workers will be spawned until that request to spawn N workers is satisfied.  If a dynamic worker cannot process a message at that time, it will sleep for 100 milliseconds. If a dynamic worker cannot process a message for 2 seconds it dies.
You may specify fixed number of workers using  `--output-http-workers=20` option.

### Follow redirects
By default Gor will ignore all redirects since they are handled by clients using your app, but in scenarios where your replayed environment introduces new redirects, you can enable them like this: 
```
gor --input-tcp replay.local:28020 --output-http http://staging.com --output-http-redirects 2
```
The given example will follow up to 2 redirects per request.

### HTTP timeouts
By default http timeout for both request and response is 5 seconds. You can override it like this:
```
gor --input-tcp replay.local:28020 --output-http http://staging.com --output-http-timeout 30s
```

### Rate limiting
Rate limiting can be useful if you only want to forward parts of production traffic and not overload your staging environment. There are 2 strategies: dropping random requests or dropping fractions of requests based on Header or URL param value. 

#### Dropping random requests
Every input and output supports random rate limiting.
There are 2 limiting algorithms: absolute or percentage based. 

Absolute: If for current second it reached specified requests limit - disregard the rest, on next second counter reseted.

Percentage: For input-file it will slowdown or speedup request execution, for the rest it will use random generator to decide if request pass or not based on chance you specified. 

You can specify your desired limit using the
"|" operator after the server address:

#### Limiting replay using absolute number
```
# staging.server will not get more than 10 requests per second
gor --input-tcp :28020 --output-http "http://staging.com|10"
```

#### Limiting listener using percentage based limiter
```
# replay server will not get more than 10% of requests 
# useful for high-load environments
gor --input-raw :80 --output-tcp "replay.local:28020|10%"
```

#### Limiting based on Header or URL param value
If you have unique user id (like API key) stored in header or URL you can consistently forward specified percent of traffic only for fraction of this users. 
Basic formula looks like this: `FNV32-1A_hashing(value) % 100 >= chance`. Examples:
```
# Limit based on header value
gor --input-raw :80 --output-tcp "replay.local:28020|10%" --http-header-limiter "X-API-KEY: 10%"

# Limit based on header value
gor --input-raw :80 --output-tcp "replay.local:28020|10%" --http-param-limiter "api_key: 10%"
```
Only percentage based limiting supported.

### Filtering 

#### Allow url regexp
```
# only forward requests being sent to the /api endpoint
gor --input-raw :8080 --output-http staging.com --http-allow-url /api
```

#### Disallow url regexp
```
# only forward requests NOT being sent to the /api... endpoint
gor --input-raw :8080 --output-http staging.com --http-disallow-url /api
```
#### Filter based on regexp of header

```
# only forward requests with an api version of 1.0x
gor --input-raw :8080 --output-http staging.com --http-allow-header api-version:^1\.0\d

# only forward requests NOT containing User-Agent header value "Replayed by Gor"
gor --input-raw :8080 --output-http staging.com --http-disallow-header "User-Agent: Replayed by Gor"
```

#### Filter based on http method
Requests not matching a specified whitelist can be filtered out. For example to strip non-nullipotent requests:

```
gor --input-raw :80 --output-http "http://staging.server" \
    --http-allow-method GET \
    --http-allow-method OPTIONS
```

### Rewriting original request
Gor supports some basic request rewriting support. For complex logic you can use middleware, see below.

#### Rewrite URL based on a mapping
```
# rewrite url to match the following
gor --input-raw :8080 --output-http staging.com --http-rewrite-url /v1/user/([^\\/]+)/ping:/v2/user/$1/ping
```

#### Set URL param
Set request url param, if param already exists it will be overwritten
```
gor --input-raw :8080 --output-http staging.com --http-set-param api_key=1
```

#### Set Header
Set request header, if header already exists it will be overwritten. This may be useful if you need to identify requests generated by Gor or enable feature flagged functionality in an application:

```
gor --input-raw :80 --output-http "http://staging.server" \
    --http-header "User-Agent: Replayed by Gor" \
    --http-header "Enable-Feature-X: true"
```

#### Host header
Host header gets special treatment. By default Host get set to the value specified in --output-http. If you manually set --http-header "Host: anonther.com", Gor will not override Host value.

If you app accepts traffic from multiple domain, and you want to keep original headers, there is specific `--http-original-host` with tells Gor do not touch Host header at all.

### Middleware
Middleware is a program that accepts request and response payload at STDIN and emits modified requests at STDOUT. You can implement any custom logic like stripping private data, advanced rewriting, support for oAuth and etc.

```
                   Original request      +--------------+
+-------------+----------STDIN---------->+              |
|  Gor input  |                          |  Middleware  |
+-------------+----------STDIN---------->+              |
                   Original response     +------+---+---+
                                                |   ^
+-------------+    Modified request             v   |
| Gor output  +<---------STDOUT-----------------+   |
+-----+-------+                                     |
      |                                             |
      |            Replayed response                |
      +------------------STDIN----------------->----+
```

Middleware can be written in any language, see `examples/middleware` folder for examples.
Middleware program should accept the fact that all communication with Gor is asynchronous, there is no guarantee that original request and response messages will come one after each other. Your app should take care of the state if logic depends on original or replayed response, see `examples/middleware/token_modifier.go` as example.

Simple bash echo middleware (returns same request) will look like this:
```bash
while read line; do
  echo $line
end
```

Middleware can be enabled using `--middleware` option, by specifying path to executable file:
```
gor --input-raw :80 --middleware "/opt/middleware_executable" --output-http "http://staging.server"
```

#### Communication protocol
All messages should be hex encoded, new line character specifieds the end of the message, eg. new message per line.

Decoded payload consist of 2 parts: header and HTTP payload, separated by new line character.  

Example request payload:

```
1 932079936fa4306fc308d67588178d17d823647c 1439818823587396305
GET /a HTTP/1.1
Host: 127.0.0.1

```

Example response payload:

```
2 8e091765ae902fef8a2b7d9dd960e9d52222bd8c 2782013
HTTP/1.1 200 OK
Date: Mon, 17 Aug 2015 13:40:23 GMT
Content-Length: 0
Content-Type: text/plain; charset=utf-8

```

Header contains request meta information separated by spaces. First value is payload type, possible values: `1` - request, `2` - original response, `3` - replayed response.
Next goes request id: unique among all requests (sha1 of time and Ack), but remain same for original and replayed response, so you can create associations between request and responses. Third argument varies depending on payload type: for request - start time, for responses - round-trip time.

HTTP payload is unmodified HTTP requests/responses intercepted from network. You can read more about request format [here](http://www.jmarshall.com/easy/http/), [here](https://en.wikipedia.org/wiki/Hypertext_Transfer_Protocol) and [here](http://www.w3.org/Protocols/rfc2616/rfc2616.html). You can operate with payload as you want, add headers, change path, and etc. Basically you just editing a string, just ensure that it is RCF compliant.

At the end modified (or untouched) request should be emitted back to STDOUT, keeping original header, and hex-encoded. If you want to filter request, just not send it. Emitting responses back is required, even if you did not touch them.

#### Advanced example
Imagine that you have auth system that randomly generate access tokens, which used later for accessing secure content. Since there is no pre-defined token value, naive approach without middleware (or if middleware use only request payloads) will fail, because replayed server have own tokens, not synced with origin. To fix this, our middleware should take in account responses of replayed and origin server, store `originalToken -> replayedToken` aliases and rewrite all requests using this token to use replayed alias. See `examples/middleware/token_modifier.go` and `middleware_test.go#TestTokenMiddleware` as example of described scheme.

### Saving requests to file and replaying them
You can save requests to file, and replay them later:
```
# write to file
gor --input-raw :80 --output-file requests.gor

# read from file
gor --input-file requests.gor --output-http "http://staging.com"
```

**Note:** Replay will preserve the original time differences between requests.

### Load testing

Currently it supported only by `input-file` and only when using percentage based limiter. Unlike default limiter for `input-file` instead of dropping requests it will slowdown or speedup request emitting. Note that unlike examples above limiter is applied to input:

```
# Replay from file on 2x speed 
gor --input-file "requests.gor|200%" --output-http "staging.com"
```

### Basic Auth

If your development or staging environment is protected by Basic Authentication then those credentials can be injected in during the replay:

```
gor --input-raw :80 --output-http "http://user:pass@staging .com"
```

Note: This will overwrite any Authorization headers in the original request.


## Stats 

Gor can report stats on the `output-tcp` and `output-http` request queues. Stats are reported to the console every 5 seconds in the form `latest,mean,max,count,count/second` by using the `--output-http-stats` and `--output-tcp-stats` options.

Examples:

```
2014/04/23 21:17:50 output_tcp:latest,mean,max,count,count/second
2014/04/23 21:17:50 output_tcp:0,0,0,0,0
2014/04/23 21:17:55 output_tcp:1,1,2,68,13
2014/04/23 21:18:00 output_tcp:1,1,2,92,18
2014/04/23 21:18:05 output_tcp:1,1,2,119,23
```

```
Version: 0.8
2014/04/23 21:19:46 output_http:latest,mean,max,count,count/second
2014/04/23 21:19:46 output_http:0,0,0,0,0
2014/04/23 21:19:51 output_http:0,0,0,0,0
2014/04/23 21:19:56 output_http:0,0,0,0,0
2014/04/23 21:20:01 output_http:1,0,1,50,10
2014/04/23 21:20:06 output_http:1,1,4,72,14
2014/04/23 21:20:11 output_http:1,0,1,179,35
2014/04/23 21:20:16 output_http:1,0,1,148,29
2014/04/23 21:20:21 output_http:1,1,2,91,18
2014/04/23 21:20:26 output_http:1,1,2,150,30
2014/04/23 21:18:15 output_http:100,99,100,70,14
2014/04/23 21:18:21 output_http:100,99,100,55,11
```

### How can I tell if I have bottlenecks?
Key areas that sometimes experience bottlenecks are the output-tcp and output-http functions which have internal queues for requests. Each queue has an upper limit of 100. Enable stats reporting to see if any queues are experiencing bottleneck behavior.
 
#### output-http bottlenecks
When running a Gor replay the output-http feature may bottleneck if:

  * the replay has inadequate bandwidth. If the replay is receiving or sending more messages than its network adapter can handle the output-http-stats  may report that the output-http queue is filling up. See if there is a way to upgrade the replay's bandwidth.
  * with `--output-http-workers` set to anything other than `-1` the `-output-http` target is unable to respond to messages in a timely manner. The http output workers which take messages off the output-http queue, process the request, and ensure that the request did not result in an error may not be able to keep up with the number of incoming requests. If the replay is not using dynamic worker scaling (`--output-http-workers=-1`)  The optimal number of output-http-workers can be determined with the formula `output-workers = (Average number of requests per second)/(Average target response time per second)`.

#### output-tcp bottlenecks
When using the Gor listener the output-tcp feature may bottleneck if:

  * the replay is unable to accept and process more requests than the listener is able generate. Prior to troubleshooting the output-tcp bottleneck, ensure that the replay target is not experiencing any bottlenecks. 
  * the replay target has inadequate bandwidth to handle all its incoming requests.  If a replay target's incoming bandwidth is maxed out the output-tcp-stats may report that the output-tcp queue is filling up. See if there is a way to upgrade the replay's bandwidth.

### ElasticSearch 
For deep response analyze based on url, cookie, user-agent and etc. you can export response metadata to ElasticSearch. See [ELASTICSEARCH.md](ELASTICSEARCH.md) for more details.

```
gor --input-tcp :80 --output-http "http://staging.com" --output-http-elasticsearch "es_host:api_port/index_name"
```

## Additional help

Feel free to ask question directly by email or by creating github issue.

## Latest releases (including binaries)

https://github.com/buger/gor/releases

## Command line reference
`gor -h` output:
```
  -http-allow-header=[]: A regexp to match a specific header against. Requests with non-matching headers will be dropped:
   gor --input-raw :8080 --output-http staging.com --http-allow-header api-version:^v1
  -http-disallow-header=[]: A regexp to match a specific header against. Requests with matching headers will be dropped:
   gor --input-raw :8080 --output-http staging.com --http-disallow-header "User-Agent: Replayed by Gor"
  -http-allow-method=[]: Whitelist of HTTP methods to replay. Anything else will be dropped:
  gor --input-raw :8080 --output-http staging.com --http-allow-method GET --http-allow-method OPTIONS
  -http-allow-url=[]: A regexp to match requests against. Filter get matched agains full url with domain. Anything else will be dropped:
   gor --input-raw :8080 --output-http staging.com --http-allow-url ^www.
  -http-disallow-url=[]: A regexp to match requests against. Filter get matched agains full url with domain. Anything else will be forwarded:
   gor --input-raw :8080 --output-http staging.com --http-disallow-url ^www.
  -http-header-limiter=[]: Takes a fraction of requests, consistently taking or rejecting a request based on the FNV32-1A hash of a specific header:
   gor --input-raw :8080 --output-http staging.com --http-header-imiter user-id:25%
  -http-param-limiter=[]: Takes a fraction of requests, consistently taking or rejecting a request based on the FNV32-1A hash of a specific GET param:
   gor --input-raw :8080 --output-http staging.com --http-param-limiter user_id:25%
  -http-rewrite-url=[]: Rewrite the request url based on a mapping:
  gor --input-raw :8080 --output-http staging.com --http-rewrite-url /v1/user/([^\/]+)/ping:/v2/user/$1/ping
  -http-set-header=[]: Inject additional headers to http reqest:
  gor --input-raw :8080 --output-http staging.com --http-set-header 'User-Agent: Gor'
  -http-set-param=[]: Set request url param, if param already exists it will be overwritten:
  gor --input-raw :8080 --output-http staging.com --http-set-param api_key=1
  -input-dummy=[]: Used for testing outputs. Emits 'Get /' request every 1s
  -input-file=[]: Read requests from file:
  gor --input-file ./requests.gor --output-http staging.com
  -input-http=[]: Read requests from HTTP, should be explicitly sent from your application:
  # Listen for http on 9000
  gor --input-http :9000 --output-http staging.com
  -input-raw=[]: Capture traffic from given port (use RAW sockets and require *sudo* access):
  # Capture traffic from 8080 port
  gor --input-raw :8080 --output-http staging.com
  -input-tcp=[]: Used for internal communication between Gor instances. Example:
  # Receive requests from other Gor instances on 28020 port, and redirect output to staging
  gor --input-tcp :28020 --output-http staging.com
  -memprofile="": write memory profile to this file
  -output-dummy=[]: Used for testing inputs. Just prints data coming from inputs.
  -output-file=[]: Write incoming requests to file:
  gor --input-raw :80 --output-file ./requests.gor
  -output-http=[]: Forwards incoming requests to given http address.
  # Redirect all incoming requests to staging.com address
  gor --input-raw :80 --output-http http://staging.com
  -output-http-elasticsearch="": Send request and response stats to ElasticSearch:
  gor --input-raw :8080 --output-http staging.com --output-http-elasticsearch 'es_host:api_port/index_name'
  -output-http-header-filter=[]: WARNING: `--output-http-header-filter` DEPRECATED, use `--http-allow-header` instead
  -output-http-redirects=0: Enable how often redirects should be followed.
  -output-http-stats=false: Report http output queue stats to console every 5 seconds. Remember to include also `--stats`
  -output-http-workers=0: Gor uses dynamic worker scaling by default.  Enter a number to run a set number of workers.
  -output-tcp=[]: Used for internal communication between Gor instances. Example:
  # Listen for requests on 80 port and forward them to other Gor instance on 28020 port
  gor --input-raw :80 --output-tcp replay.local:28020
  -output-tcp-stats=false: Report TCP output queue stats to console every 5 seconds. Remember to include also `--stats`
  -split-output=false: By default each output gets same traffic. If set to `true` it splits traffic equally among all outputs.
  -stats=false: Turn on queue stats output. Use in combination with the other *-stats flags.
  -verbose=false: Turn on verbose/debug output
```

## Building from source

1. Setup standard Go environment http://golang.org/doc/code.html and ensure that $GOPATH environment variable properly set.
2. `go get github.com/buger/gor`.
3. `cd $GOPATH/src/github.com/buger/gor`
4. `go build` to get binary, or `go test` to run tests

## Development
Project contains Docker environment.

1. Build container: `make dbuild`
2. Run all tests: `make dtest`. Run specific test: `make dtest ARGS=-test.run=**regexp**`
3. Bash access to container: `make dbash`. Inside container you have python to run simple web server `python -m SimpleHTTPServer 8080` and `curl` to make http requests. 


## Questions and support 

All bug-reports and suggestions should go though Github Issues or our [Google Group](https://groups.google.com/forum/#!forum/gor-users). Or you can just send email to gor-users@googlegroups.com

If you have some private questions you can send direct mail to leonsbox@gmail.com

## FAQ

### What OS are supported?
For now only Linux based. *BSD (including MacOS is not supported yet, check https://github.com/buger/gor/issues/22 for details)

### Why does the `--input-raw` requires sudo or root access?
Listener works by sniffing traffic from a given port. It's accessible
only by using sudo or root access.

### How do you deal with user session to replay the traffic correctly?
You can rewrite session related headers/params to match your staging environment. If you require custom logic (e.g random token based auth) follow this discussion: https://github.com/buger/gor/issues/154

### Can i use Gor to intercept SSL traffic?
Basic idea is that SSL was made to protect itself from traffic interception. There 2 options: 
1. Move SSL handling to proxy like Nginx or Amazon ELB. And allow Gor to listen on upstreams. 
2. Use `--input-http` so you can duplicate request payload directly from your app to Gor, but it will require your app modifications.

More can be find here: https://github.com/buger/gor/issues/85

### Is there a limit for size of HTTP request when using output-http?
Due to the fact that Gor can't guarantee interception of all packets, for large payloads > 200kb there is chance of missing some packets and corrupting body. Treat it as a feature and chance to test broken bodies handling :)
The only way to guarantee delivery is using `--input-http`, but you will miss some features.

### I'm getting 'too many open files' error
Typical linux shell has a small open files soft limit at 1024. You can easily raise that when you do this before starting your gor replay process:
  
  ulimit -n 64000

More about ulimit: http://www.thecodingmachine.com/solving-the-too-many-open-files-exception-in-red5-or-any-other-application/

### The CPU average across my load-balanced targets is higher than the source
If you are replaying traffic from multiple listeners to a load-balanced target and you use sticky sessions, you may observe that the target servers have a higher CPU load than the listener servers. This may be because the sticky session cookie of the original load balancer is not honored by the target load balancer thus resulting in requests that would normally hit the same target server hitting different servers on the backend thus reducing some caching benefits gained via the load balancing.  Try running just one listener against one replay target and see if the CPU utilization comparison is more accurate.

## Tuning

To achieve the top most performance you should tune the source server system limits:

    net.ipv4.tcp_max_tw_buckets = 65536
    net.ipv4.tcp_tw_recycle = 1
    net.ipv4.tcp_tw_reuse = 0
    net.ipv4.tcp_max_syn_backlog = 131072
    net.ipv4.tcp_syn_retries = 3
    net.ipv4.tcp_synack_retries = 3
    net.ipv4.tcp_retries1 = 3
    net.ipv4.tcp_retries2 = 8
    net.ipv4.tcp_rmem = 16384 174760 349520
    net.ipv4.tcp_wmem = 16384 131072 262144
    net.ipv4.tcp_mem = 262144 524288 1048576
    net.ipv4.tcp_max_orphans = 65536
    net.ipv4.tcp_fin_timeout = 10
    net.ipv4.tcp_low_latency = 1
    net.ipv4.tcp_syncookies = 0


## Contributing

1. Fork it
2. Create your feature branch (git checkout -b my-new-feature)
3. Commit your changes (git commit -am 'Added some feature')
4. Push to the branch (git push origin my-new-feature)
5. Create new Pull Request

## Companies using Gor

* [Granify](http://granify.com)
* [GOV.UK](https://www.gov.uk) ([Government Digital Service](http://digital.cabinetoffice.gov.uk/))
* [theguardian.com](http://theguardian.com)
* [TomTom](http://www.tomtom.com/)
* [3SCALE](http://www.3scale.net/)
* [Optionlab](http://www.opinionlab.com)
* [TubeMogul] (http://tubemogul.com)
* To add your company drop me a line to github.com/buger or leonsbox@gmail.com
