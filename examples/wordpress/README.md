### Wordpress on GCE

Crate your cluster if you have not done so already:

```
$ gcloud container clusters create wordpress
```

Create a persistant storage disk for MySQL, called `mysql-disk`:

```
$ gcloud compute disks create mysql-disk
```

Deploy your containers:

```
$ kubectl create -f mysql.json
$ kubectl create -f mysql-service.json
$ kubectl create -f wordpress-rc.json
$ kubectl create -f wordpress-service.json
```

Check your pods and services:

```
$ kubectl get pods
$ kubectl get services
```

After a few moments your load balancer should have been created. You can check
up on that like so:

```
$ kubectl describe services wordpress-service
```

Look for the "LoadBalancer Ingress" key. You can do this from the output of the
above command or by using grep to just get the IP:

```
$ kubectl describe services wordpress-service | grep "LoadBalancer Ingress"
```

Head over to that IP address in your web browser to go through the Wordpress
installation.

And you're done!

##### Next

Okay so the above will have set your wordpress site up. There are things you
will want to do to tweak this though. The disk that we initiall set up will be
500GB by default. You should change this to suit your needs.

The same goes for creating your cluster. This uses all the default settings.
While this will work just fine, you should refine it to work for you.

As for the JSON documents, these will also work, but use `root` and `password`
for your database, so this would need to be changed for your own site. Along
with that are a bunch of other environment variables that can be found on
the Wordpress Docker Hub page, the same goes for MySQL.

Although `wordpress.json` is not used in this example, you may use it instead of
the replication controller if you wish. The choice is yours.
