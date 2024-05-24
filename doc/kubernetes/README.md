# Kubernetes Deployments

In this folder you will find some Kubernetes manifests examples to deploy the Integration Test WebServer in a Kubernetes cluster.

:warning: Please remember that even though this server offers basic auth and TLS capabilities, it **SHOULD NOT** be accessible from the Internet.

## deployment.yml

This files contains a simple deployment example for the Integration toolbox webserver. It comes with a config map referrencing all possible environments variables to configure the server, set to their default values.

The deployment has some resources requests and limits defined but commented out. Feel free to un-comment and modify them. Default values should be sufficient for most use cases.

Once the deployment checked and eventually modified, you can deploy the Integration Toolbox WebServer using this simple command:

```bash
kubectl apply -f deployment.yml
```

When the pod is up and running, to access the web interface run:

```bash
kubectl port-forward pod/name-of-the-itw-pod 8080:8080
```

and then connect to [http://localhost:8080/ui/](http://localhost:8080/ui/) with your browser.

## hpa.yml

This files defines a Kubernetes HorizontalPodAutoscaler to enable auto-scaling of the Intergration Toolbox WebServer. The auto-scalling is configured to watch CPU resources (a section is commented with memory metrics).

For it to work, you first need to make sure that your have defined container resources in the deployment (commented out in the deployment manifest).

To deploy:

```bash
kubectl apply -f hpa.yml
```

To see it in action, you can call the [/cpu/load](../../README.md#cpuload) endpoint to increase the CPU load.

## ingress.yml

This file contains an ingress manifest if you want to make the server accessible from the ingress (internal ingress recommanded). At minimum, you will to configure the host name, and the ingress class name before deploying.

To deploy:

```bash
kubectl apply -f ingress.yml
```

Once deployed, make sure you have you DNS correctly configured then you'll be able to access it from you browser, at the URL configured in the ingress.