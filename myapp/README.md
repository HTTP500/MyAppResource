### Step
1. Install dependencies: minikube, make, golang, kubectl
1. Start minikube and clone the repo
1. Under myapp/config/bases, use `kubectl apply -f ./crd-myappresource.yaml` to add the CRD
1. under myapp/ , get all golang dependencies then use `make run` to build and run the controller
1. Under myapp/config/bases, use `kubectl apply -f ./target` (which is the given CR in the homework)
1. Use `minikube tunnel` to expose the LoadBalancer then you will be able to use `kubectl get svc` to get the exposed IP. From that IP you will be able to access the podinfo service

### Limitations (Due to time constraint, i.e. ~9h)
1. We do not handle resource deletion yet
1. Project folder needs clean up (generated by kubebuild)
1. There is no test