### Jenkins on GCE

This is a very basic guide that I'll keep coming back to with improvements, but
it should give you a start with getting Jenkins running on GCE.

Crate your cluster if you have not done so already:

```
$ gcloud container clusters create jenkins
```

Deploy your containers:

```
$ kubectl create -f jenkins.json
$ kubectl create -f jenkins-service.json
```

Check your pods and services:

```
$ kubectl get pods
$ kubectl get services
```

Wait for the load balancer to be created. You can keep checking for the public
IP by doing the following:

```
$ kubectl describe services jenkins-service | grep "LoadBalancer Ingress"
```

Head over to that IP address in your web browser.

Done!
