# An overview of Kubernetes

Recently I have been playing around with Kubernetes on Google Container Engine.
At first it seemed pretty daunting to me and from what I've heard from a few
other people it has been the same for them. But once you start getting used to
how Kubernetes works and how you're supposed to use it, you realise how powerful
it can be and how it can make your deployments seem almost effortless. The goal
of this document is to go through Kubernetes step by step in such a way whereby
the only prerequisite for you is that you understand what containers are and
ideally, how Docker works. This is just Kubernetes in my own words. I hope you
find it helpful. (WIP)

### So what is it?
In short, Kubernetes is a set of tools and programs that give you higher level
control of your cluster and everything running on it. Once Kubernetes is all set
up on your cluster you can start pods, services, and have your containers all
running in harmony. You can find out more about it, along with Google Container
Engine [here][1].


### Up and running quickly
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

### The application

First, we need an application that we'd like to run on our cluster. We could
use one that already exists, but for this introduction I'm going to create one
using the [Go programming language][4].


`hello.go`
```go
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

### Starting the cluster

Now we have our container image pushed to the Docker hub, we can get a start on
creating the cluster.

Once you have gcloud installed, you'll be able to install kubectl:

```
$ gcloud components update kubectl
```

If you've set up Google Container Engine and have enabled billing etc... you
should have a default project listed. You can access that [here][5].

NOTE: If you've not done this yet and feel uneasy about paying for something you
might never use, Google (at least as of writing this) offer 60 days and $300
worth of resources to use with Google Cloud Platform when you initially sign up,
no strings attached. This is a total steal, just [go here][7] then click on
Free Trial.

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

### Pods

This is where things might start to seem a little daunting if you're new to
Kubernetes. But once you get used to it, it's really not that bad. Kubernetes
has this concept of pods. Pods are basically a collection of 1 or more
containers, it's that simple. Within a pod you can allocate resources and limits
to what the containers have access to.

Kubernetes makes it easy for us to create multiple pods of the same application
and sit them behind a load balancer. We can also tell Kubernetes that at anytime
we'd like 3 instances of our app to be running. Or 4, or 5. It doesn't really
matter too much at this point. What does matter though, is that Kubernetes will
make sure that how ever many you specify to be running at one time, it will
stick to that, by means of creating new pods if one were to go down, or to bring
a pod down if at anytime there are too many running. I'll get onto that shortly.

First though, let's create a single pod. We'll do this by use of a JSON
document. You may also use YAML if you prefer that.

`hello-pod.json`
```json
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

As you can see there is also a metadata section, where we can name this pod,
and provide it some key / value pairs under labels. Labels, as you'll come to
see are pretty important, as we can select resources to be ran by using them,
as opposed to just a name.

And finally the `spec`, which is where we list our containers to be ran in this
pod. We're only going to be using the one. This will be pulled from the Docker
registry that we pushed to earlier on.

We then specify the ports, and if you remember from earlier our Go application
runs on port 3000, so we'll use that here also.

Let's now create this pod:

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

### Services

So what is a service and why do we need them? As far as Kubernetes is concerned,
a service is basically a named load balancer. In essence this means that we
could have multiple pods running on our cluster, and have our service use the
metadata to select the pods that are relevant. You will then be able to hit the
application that is running inside one of your pods via the public IP of the
service.

So let's create the service:

`hello-service.json`
```json
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
`hello`. The target port of this pod is 3000, as specified in the pod file, but
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

It may take a few moments, but once the service is up and running, you should be
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

### Replication Controllers

A replication controller is responsible for keeping a defined amount of pods
running at any given time. You may have your replication controller create 4
pods for example. If one goes down, the controller will be sure to start another
one. If for some reason a 5th one appears, it will kill one to bring it back
down to 4. It's a pretty straight-forward concept, and better yet we can make
use of our pod files we made earlier to create our first replication
controller.

`hello-rc.json`
```json
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

So if you look beyond `template`, you'll see the same pod that we created
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

### Scaling

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

### Persistent Storage

NOTE: To go through this next part, you may want to delete all the pods and
services currently running, first:

```
$ kubectl delete services hello-service
$ kubectl delete rc hello-rc
```

Something your application may need is storage. You may be dealing with file
uploads, or a database. Although containers should be seen as being fairly
disposable, your storage / data should remain. GCE makes this really easy for us
as all we need to do is to create a disk, and then tell our containers where
that disk is.

So let's make a disk first of all:

```
$ gcloud compute disks create mysql-disk
```

By default, this will create a 500GB disk. You can change this among various
other settings by passing flags to that command.

Within the scope of `containers` in either your pod or ReplicationController
file, you can add another section called `volumeMounts`. Here, you are able to
specify where inside your container you'd like to mount.

I've created a new application called todo.go. It is based off our original
hello.go application, only it actually does something this time:

`todo.go`
```go
package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
)

const dbHost = "mysql-service:3306"
const dbUser = "root"
const dbPassword = "password"
const dbName = "todo"

func main() {
	http.HandleFunc("/", todoList)
	http.HandleFunc("/save", saveItem)
	log.Fatal(http.ListenAndServe(":3000", nil))
}

// Todo represents a single 'todo', or item.
type Todo struct {
	ID   int
	Item string
}

// todoList shows the todo list along with the form to add a new item to the
// list.
func todoList(w http.ResponseWriter, r *http.Request) {
	tmpl := `
<html>
	<head>
		<title>List of items</title>
	</head>
	<body>
		<h1>Todo list:</h1>
		<form action="/save" method="post">
			Add item: <input type="text" name="item" /><br />
			<input type="submit" value="Add" /><br /><hr />
		</form>
		<ul>
		{{range $key, $value := .TodoItems}}
			<li>{{ $value.Item }}</li>
		{{end}}
		</ul>
	</body>
</html>
`
	t, err := template.New("todolist").Parse(tmpl)
	if err != nil {
		log.Fatal(err)
	}

	dbc := db()
	rows, err := dbc.Query("SELECT * FROM items")
	if err != nil {
		log.Fatal(err)
	}

	todos := []Todo{}
	for rows.Next() {
		todo := Todo{}
		err = rows.Scan(&todo.ID, &todo.Item)
		todos = append(todos, todo)
	}

	data := struct {
		TodoItems []Todo
	}{
		TodoItems: todos,
	}

	t.Execute(w, data)
}

// saveItem saves a new todo item and then redirects the user back to the list
func saveItem(w http.ResponseWriter, r *http.Request) {
	dbc := db()
	stmt, err := dbc.Prepare("INSERT items SET item=?")
	if err != nil {
		log.Fatal(err)
	}
	_, err = stmt.Exec(r.FormValue("item"))
	if err != nil {
		log.Fatal(err)
	}
	http.Redirect(w, r, "/", 301)
}

// db creates a connection to the database and creates the items table if it
// does not already exist.
func db() *sql.DB {
	connStr := fmt.Sprintf(
		"%s:%s@tcp(%s)/%s?charset=utf8&parseTime=True&loc=Local",
		dbUser,
		dbPassword,
		dbHost,
		dbName,
	)
	db, err := sql.Open("mysql", connStr)
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS items(
		id integer NOT NULL AUTO_INCREMENT,
		item varchar(255),
		PRIMARY KEY (id)
	)`)
	return db
}

```

Now we can go through similar steps from earlier to create our binary and Docker
container image. This one will be called `todo`.

```
$ go get -u github.com/go-sql-driver/mysql
$ CGO_ENABLED=0 GOOS=linux go build -o todo -a -tags netgo -ldflags '-w' .
```

`Dockerfile`
```
FROM scratch
ADD todo /
CMD ["/todo"]
```

```
$ docker build -t DOCKERHUB_USERNAME/todo:latest .
$ docker push DOCKERHUB_USERNAME/todo
```

That is our container image pushed to the Docker Hub. Now onto defining our
replication controller and services. Seeing as we'll be using MySQL this time,
we'll be creating a seperate disk using the gcloud cli tool. This is what we'll
use when configuring the `volumes` for our containers.

```
$ gcloud compute disks create mysql-disk
```

`mysql-disk` is going to be the name of it. This is important as it is used
to reference the disk in our JSON files. Once that is done, we can go ahead and
create the MySQL pod.

`mysql.json`
```json
{
  "apiVersion": "v1",
  "kind": "Pod",
  "metadata": {
    "name": "mysql",
    "labels": {
      "name": "mysql"
    }
  },
  "spec": {
    "containers": [
      {
        "name": "mysql",
        "image": "mysql:5.6",
        "env": [
          {
            "name": "MYSQL_ROOT_PASSWORD",
            "value": "password"
          },
          {
            "name": "MYSQL_DATABASE",
            "value": "todo"
          }
        ],
        "ports": [
          {
            "name": "mysql",
            "protocol": "TCP",
            "containerPort": 3306
          }
        ],
        "volumeMounts": [
          {
            "name": "mysql-storage",
            "mountPath": "/var/lib/mysql"
          }
        ]
      }
    ],
    "volumes": [
      {
        "name": "mysql-storage",
        "gcePersistentDisk": {
          "pdName": "mysql-disk",
          "fsType": "ext4"
        }
      }
    ]
  }
}

```

As you can see in this file, we specify a few more things than in our original
hello pod. We'll be using the official `mysql:5.6` container image from Docker
Hub. Environment variables are set to configure various aspects of how this pod
will run. You can find documentation for these at the Docker Hub on the MySQL
page.

We'll just be setting the basics, `MYSQL_ROOT_PASSWORD` and `MYSQL_DATABASE`.
This will give us access to our `todo` database as the `root` user. The next new
thing here are the `volumeMounts` and `volumes` keys. We give each volume mount
a name and path. The name is what is then used by the `volumes` as a reference
to it. The mount path is just where on the container you'll mount to.

Outside of the scope of our containers, we can specify where our volume mounts
will be. In this case, we'll be using the newly created `mysql-disk` from
earlier, and defining the file system type to `ext4`. Now we can start the pod:

```
$ kubectl create -f mysql.json
```

Next, we'll create a service for this pod. Like earlier, our service is going to
be acting as a load balancer. We'll say what port we'd like to listen on, along
with the target port of the running container:

`mysql-service.json`
```
{
  "apiVersion": "v1",
  "kind": "Service",
  "metadata": {
    "name": "mysql-service",
    "labels": {
      "name": "mysql"
    }
  },
  "spec": {
    "selector": {
      "name": "mysql"
    },
    "ports": [
      {
        "protocol": "TCP",
        "port": 3306,
        "targetPort": 3306
      }
    ]
  }
}
```

We can now start this:

```
$ kubectl create -f mysql-service.json
```

Take note of the `name` key in this file. Google Container Engine comes with
a DNS service already running, which means pods are able to access each other
using the value to `name` as the host. If you noticed in our Go program, we
specify the host as `mysql-service:3306`. This can also be done with environment
variables. I'll go into detail with that another time.

With MySQL running, we should be able to start our todo application now. For
this example I'll be using a replication controller rather than just a single
pod definition:

`todo-rc.json`
```json
{
  "apiVersion": "v1",
  "kind": "ReplicationController",
  "metadata": {
    "name": "todo-rc",
    "labels": {
      "name": "todo-rc"
    }
  },
  "spec": {
    "replicas": 3,
    "selector": {
      "name": "todo"
    },
    "template": {
      "metadata": {
        "name": "todo",
        "labels": {
          "name": "todo"
        }
      },
      "spec": {
        "containers": [
          {
            "name": "todo",
            "image": "DOCKERHUB_USERNAME/todo:latest",
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

You'll notice this file is very similar to our hello app replication controller.
We can start this now:

```
$ kubectl create -f todo-rc.json
```

And the service, with `"type": "LoadBalancer"` to expose a public IP:

`todo-service.json`
```json
{
  "apiVersion": "v1",
  "kind": "Service",
  "metadata": {
    "name": "todo-service",
    "labels": {
      "name": "todo-service"
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
      "name": "todo"
    }
  }
}
```

The same as always:

```
$ kubectl create -f todo-service.json
```

Once the service has finished creating the load balancer, you can head over to
the public IP to see your application running.

More coming soon...

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
[7]: https://cloud.google.com/

