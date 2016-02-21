# An overview of Kubernetes

Recently I have been playing around with Kubernetes on Google Container Engine.
At first it seemed pretty daunting to me, but after a while I started getting
used to it and realised how powerful it is. This document is just me explaining
it in my own words as I'm using it. It is a work in progress.

#### So what is it?
In short, Kubernetes is a set of tools and programs that give you higher level
control of your cluster and everything running on it. Once Kubernetes is all set
up on your cluster you can start pods, services, and have your containers all
running in harmony. You can find out more about it, along with Google Container
Engine [here][1].


#### Up and running quickly
A great way to get up and running quickly with Kubernetes is to get set up with
Google Container Engine, which allows you to start up a cluster with everything
working from the get go. You can manage aspects of this using the [gcloud][2]
cli tool, which I'll be using in this brief introduction. You can also get up
and running locally using Vagrant, or elsewhere as listed on the
[getting started page][3].

For this tutorial I will be using the following:

  - Go
  - Docker / Docker Hub
  - A Linux environment
  - gcloud cli application /w kubectl
  - Google Container Engine / Kubernetes

#### The application

First, we need an application that we'd like to run on our cluster. We could
use one that already exists, but for this introduction I'm going to create one
using the [Go programming language][4].


`hello.go`
```
package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(":3000", nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello from Go!")
}
```

Next, we'll compile this program making sure it is statically linked:

Go 1.4:

```
$ CGO_ENABLED=0 GOOS=linux go build -o hello -a -installsuffix cgo .
```

Go 1.5:

```
$ CGO_ENABLED=0 GOOS=linux go build -o hello -a -tags netgo -ldflags '-w' .
```

We're now left with our `hello` binary. If you're already on Linux, you'll be
able to test this. If not, you can boot up a VM and try it out there. We'll be
using Docker next to create our container image.

As we already have our compiled application, our Dockerfile is going to be super
simple:

`Dockerfile`
```
FROM scratch
ADD hello /
CMD ["/hello"]
```

We can then build this and push it to the Docker registry:

```
$ docker build -t hello .
$ docker images
$ docker tag IMAGE_ID DOCKERHUB_USERNAME/hello:latest
$ docker push DOCKERHUB_USERNAME/hello
```

So now we have our image built and pushed, we should be able to test running it:

```
$ docker run -p 3000:3000 DOCKERHUB_USERNAME/hello
```

And to check that our application is working correctly:

```
$ curl localhost:3000
Hello from Go!
```

#### Starting the cluster

Now we have our container image pushed to the Docker hub, we can get a start on
creating the cluster.

Once you have gcloud installed, you'll be able to install kubectl:

```
$ gcloud components update kubectl
```

If you've set up Google Container Engine and have enabled billing etc... you
should have a default project listed. You can access that [here][5].

Get the ID of the project and configure like so:

```
$ gcloud config set project PROJECT_ID
```

Next, set your default zone (you can learn about that [here][6]):

```
$ gcloud config set compute/zone ZONE
```

You can now start the cluster. I'll just be using the default settings here, but
you can use flags to customise various aspects of it.

```
$ gcloud container clusters create helloapp
```

After a few moments, your cluster will be created! You should see something like
this:

```
NAME      ZONE            MASTER_VERSION  MASTER_IP       MACHINE_TYPE   NODE_VERSION  NUM_NODES  STATUS
helloapp  europe-west1-b  1.1.7           10.10.10.10     n1-standard-1  1.1.7         3          RUNNING
```

Now we're ready to start deploying our application.

#### Pods

This is where things might start to seem a little daunting if you're new to
Kubernetes. But once you get used to it, it's really not that bad. Kubernetes
has this concept of Pods. Pods are basically a collection of 1 or more
containers, it's that simple. Within a Pod you can allocate resources and limits
to what the containers have access to.

Kubernetes makes it easy for us to create multiple Pods of the same application
and sit them behind a load balancer. We can also tell Kubernetes that at anytime
we'd like 3 instances of our app to be running. Or 4, or 5. It doesn't really
matter too much at this point. What does matter though, is that Kubernetes will
make sure that how ever many you specify to be running at one time, it will
stick to that, by means of creating new pods if one were to go down, or to bring
a pod down if at anytime there are too many running. I'll get onto that shortly.

First though, let's create a single Pod. We'll do this by use of a JSON
document. You may also use YAML if you prefer that.

`hello-pod.json`
```
{
	"apiVersion": "v1",
	"kind": "Pod",
	"metadata": {
		"name": "hello",
		"labels": {
			"name": "hello"
		}
	},
	"spec": {
		"containers": [
			{
				"name": "hello",
				"image": "DOCKERHUB_USERNAME/hello:latest",
				"ports": [
					{
						"name": "http",
						"containerPort": 3000,
						"protocol": "TCP"
					}
				]
			}
		]
	}
}

```

Let's go over this a little. So we're specifying the API version we'll be using
at the top here. In this example this is simply just `v1`. We need to tell
Kubernetes what it is we'd like to create, in this case it's a `Pod`, which is
the value to `kind`.

As you can see there is also a metadata section, where we can name this Pod,
and provide it some key / value pairs under labels. Labels, as you'll come to
see are pretty important, as we can select resources to be ran by using them,
as opposed to just a name.

And finally the `spec`, which is where we list our containers to be ran in this
Pod. We're only going to be using the one. This will be pulled from the Docker
registry that we pushed to earlier on.

We then specify the ports, and if you remember from earlier our Go application
runs on port 3000, so we'll use that here also.

Let's now create this Pod:

```
$ kubectl create -f hello-pod.json
pod "hello" created
```

If we list our pods, the newly created one should be there:

```
$ kubectl get pods
NAME      READY     STATUS    RESTARTS   AGE
hello     1/1       Running   0          1m
```

So now that is up and running, we can move on to services.

#### Services

So what is a service and why do we need them? As far as Kubernetes is concerned,
a service is basically a named load balancer. In essence this means that we
could have multiple pods running on our cluster, and have our service use the
metadata to select the pods that are relevant. You will then be able to hit the
application that is running inside one of your pods via the public IP of the
service.

So let's create the service:

`hello-service.json`
```
{
	"apiVersion": "v1",
	"kind": "Service",
	"metadata": {
		"name": "hello-service",
		"labels": {
			"name": "hello-service"
		}
	},
	"spec": {
		"type": "LoadBalancer",
		"ports": [
			{
				"name": "http",
				"port": 80,
				"targetPort": 3000
			}
		],
		"selector": {
			"name": "hello"
		}
	}
}
```

You can see here that we're using the `selector` key to find anything using the
specified metadata key / value pairs. In this case, something with the `name`,
`hello`. The target port of this Pod is 3000, as specified in the Pod file, but
we would like to run that on port 80, so it's more accessible. With this kind of
service, the `type` needs to be specified as `LoadBalancer`. This allows it to
be publically accessible to the outside world via IP address.

So with both our `hello-pod.json` and our `hello-service.json`, we're now able
to create them on the cluster:

```
$ kubectl create -f hello-service.json
service "hello-service" created
```

We can also list our services, and should see this new one appear:

```
$ kubectl get services
NAME            CLUSTER_IP       EXTERNAL_IP   PORT(S)   SELECTOR     AGE
hello-service   10.143.241.227                 80/TCP    name=hello   44s
kubernetes      10.143.240.1     <none>        443/TCP   <none>       8m
```

If may take a few moments, but once the service is up and running, you should be
able to visit the external IP address and see the application up and running.
You can find out what that is by running the following:

```
$ kubectl describe services hello-service
```

It should look a little like this:

```
Name:			hello-service
Namespace:		default
Labels:			name=hello-service
Selector:		name=hello
Type:			LoadBalancer
IP:			    10.143.241.227
LoadBalancer Ingress:	104.155.115.84
Port:			http	80/TCP
NodePort:		http	30962/TCP
Endpoints:		10.140.1.3:3000
Session Affinity:	None
```

If you try going to the IP address listed as `LoadBalancer Ingress:` in your
web browser, you should see the application running live.

```
$ curl 104.155.115.84
Hello from Go!
```

#### Replication Controllers

A replication controller is responsible for keeping a defined amount of pods
running at any given time. You may have your replication controller create 4
pods for example. If one goes down, the controller will be sure to start another
one. If for some reason a 5th one appears, it will kill one to bring it back
down to 4. It's a pretty straight-forward concept, and better yet we can make
use of our pod files we made earlier to create our first replication
controller.

`hello-rc.json`
```
{
	"apiVersion": "v1",
	"kind": "ReplicationController",
	"metadata": {
		"name": "hello-rc",
		"labels": {
			"name": "hello-rc"
		}
	},
	"spec": {
		"replicas": 3,
		"selector": {
			"name": "hello"
		},
		"template": {
			"metadata": {
				"name": "hello",
				"labels": {
					"name": "hello"
				}
			},
			"spec": {
				"containers": [
					{
						"name": "hello",
						"image": "DOCKERHUB_USERNAME/hello:latest",
						"ports": [
							{
								"name": "http",
								"containerPort": 3000,
								"protocol": "TCP"
							}
						]
					}
				]
			}
		}
	}
}
```

So if you look beyond `template`, you'll see the same Pod that we created
earlier on. If the replication controller is going to start and maintain a
number of pods, it needs a template to know what our pods will look like.

Above `template`, you'll see things specific to the controller itself. Notably,
the `replicas` key which defines how many of the pods specified below should
be running. It also uses a selector, which is again linked with the metadata
provided inside the pod. So this replication controller is going to look for
anything with `name` as `hello`. Above that you'll also notice keys which so
fer we've specified in each of our other files too.

So at this point we can kill the pod we created earlier on:

```
$ kubectl delete pods hello
```

and then create our replication controller:

```
$ kubectl create -f hello-rc.json
```

After that has started, you should be able to see 3 of our `hello` pods running:

```
$ kubectl get pods
```

The `hello-service` should also still be running, serving incoming request to
each of your newly created pods. We can see the replication controller in action
if we manually try to delete a pod:

```
$ kubectl delete pods POD_ID
```

and then quickly run:

```
$ kubectl get pods
```

To see what is going on under the hood.

##### Scaling

So with our replication controller handling our pods and our service balancing
the load, we can now think about scaling. As of now, we haven't specified any
limits for the pods, I will come to that shortly. So let's create a scenario:

You deploy your application. You write a blog post about it and send out a few
tweets. The traffic is on the rise. Your app is beginning to struggle with the
load. We need to create more pods to even the load out on the cluster. We're
going to do this with just one simple command and change from having 3 pods to
5 pods:

```
$ kubectl scale --replicas=5 -f hello-rc.json
```

You should now be able to run:

```
$ kubectl get pods
```

to see the two newly created ones:

```
$ kubectl get pods
NAME             READY     STATUS    RESTARTS   AGE
hello-zndh4s     1/1       Running   0          21m
hello-j8sdgd     1/1       Running   0          21m
hello-8sjn4h     1/1       Running   0          21m
hello-ka98ah     1/1       Running   0          1m
hello-qjwh37     1/1       Running   0          1m
```

This works both ways. Eventually the traffic might subside, and you can then
bring it back down to 3 pods.

```
$ kubectl scale --replicas=3 -f hello-rc.json
```

#### Persistent Storage

Coming soon...

### Questions and contributions

If you have any questions, feel free to open an issue. Contributions are more
than welcome, especially if you think there is something that can be explained
more clearly or see any mistakes.

### License

This project uses the MIT license.

[1]: https://cloud.google.com/container-engine/docs/
[2]: https://cloud.google.com/sdk/gcloud/
[3]: http://kubernetes.io/gettingstarted/
[4]: http://golang.org
[5]: https://console.cloud.google.com/project
[6]: https://cloud.google.com/compute/docs/zones#available

