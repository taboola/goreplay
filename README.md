## About

Gor is simple http traffic replication tool written in Go. 
Its main goal to replay traffic from production servers to staging and dev environments.


Now you can test your code on real user sessions in an automated and repeatable fashion.  
**No more falling down in production!**

Gor consists of 2 parts: listener and replay servers.

Listener catch http traffic from given port in real-time and send it to replay server via UDP. 
Replay server forwards traffic to given address.

## Basic example

```bash
# Run on server you want to catch traffic. You can run it on all `web` machines.
sudo gor listen -p 80 -r replay.server.local:28020 

# Replay server (replay.server.local). 
gor replay -f http://staging.server
```

## Advanced use

### Rate limiting
Replay server support rate limiting. It can be useful if you want forward only part of production traffic, not to overload staging environment. You can specify desired request per second using "|" operator after server address:

```
# staging.server not get more than 10 requests per second
gor replay -f "http://staging.server|10"
```

### Forward to multiple addresses

You can forward traffic to multiple endpoints. Just separate addresses by coma.
```
gor replay -f "http://staging.server|10"
```

## Additional help
```
$ gor listen -h
Usage of ./bin/gor-linux:
  -i="any": By default it try to listen on all network interfaces.To get list of interfaces run `ifconfig`
  -p=80: Specify the http server port whose traffic you want to capture
  -r="localhost:28020": Address of replay server.
```

```
$ gor replay -h
Usage of ./bin/gor-linux:
  -f="http://localhost:8080": http address to forward traffic.
	You can limit requests per second by adding `|#{num}` after address.
	If you have multiple addresses with different limits. For example: http://staging.example.com|100,http://dev.example.com|10
  -ip="0.0.0.0": ip addresses to listen on
  -p=28020: specify port number
```

## FAQ

### Why `gor listener` requires sudo or root access?
Listener works by sniffing traffic from given port. Its accessible only using sudo or root access.

### Do you support all http request types?
Right now it support only "GET" requests.
