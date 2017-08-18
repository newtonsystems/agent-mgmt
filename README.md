# agent-mgmt


[![](https://images.microbadger.com/badges/image/newtonsystems/tools-agent-mgmt:0.2.2.svg)](https://microbadger.com/images/newtonsystems/tools-agent-mgmt:0.2.2 "Get your own image badge on microbadger.com")

[![](https://images.microbadger.com/badges/version/newtonsystems/tools-agent-mgmt:0.2.2.svg)](https://microbadger.com/images/newtonsystems/tools-agent-mgmt:0.2.2 "Get your own version badge on microbadger.com")

Available from docker hub as [newtonsystems/tools/agent-mgmt](https://hub.docker.com/r/newtonsystems/tools-agent-mgmt/)

#### Supported tags and respective `Dockerfile` links

-    [`v0.2.2`, `v0.2.1`, `v0.2.0`, `latest` (/Dockerfile*)](https://github.com/newtonsystems/devops/blob/master/tools/agent-mgmt/Dockerfile)

# What is agent-mgmt?

A base docker image to be used for circleci for compiling and building grpc services.


## How to use with circleci

- Example curl command

```bash
curl -H "Content-Type: application/json" -X POST -d '{"Name":"abc"}' http://`minikube ip`:32000/sayhello
```


curl -H "Content-Type: application/json" -X POST -d '{"Name":"abc"}' http://`minikube ip`:32000/sayhello 


## How to test a localhost with the outside work 

Sometimes this service will need to connect to the outside world when working locally for testing etc.

We use ngrok to create secure tunnels to the localhost

Once you have installed ngrok:

```bash
ngrok http localhost:50000
```



## How to do a release
- Make sure you are using docker-utils 
i.e.

```bash
export PATH="~/<LOCATION>/docker-utils/bin:$PATH"
```

```
build-tag-push-dockerfile.py  --image "newtonsystems/tools-agent-mgmt" --version 0.1.0 --dockerhub_release --github_release
```


## Future

- Use docker when in development 
