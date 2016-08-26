## Run gitlab-runner on a Kubernetes cluster

To get started with the GitLab CI Runner on Kubernetes you need to define
resources that you can then push to the cluster with kubectl.

A recommended approach to this is to create a `ConfigMap` in Kubernetes such as
the following:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: gitlab-runner
  namespace: gitlab
data:
  config.toml: |
    concurrent = 4

    [[runners]]
      name = "Kubernetes Runner"
      url = "https://gitlab.com/ci"
      token = "...."
      executor = "kubernetes"
      [runners.kubernetes]
        namespace = "gitlab"
        privileged = true
```

Then create a `Deployment` or `ReplicationController` which uses the `ConfigMap`.
This is an example of a `Deployment`:

```yaml
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: gitlab-runner
  namespace: gitlab
spec:
  replicas: 1
  selector:
    matchLabels:
      name: gitlab-runner
  template:
    metadata:
      labels:
        name: gitlab-runner
    spec:
      containers:
      - args:
        - run
        image: gitlab/gitlab-runner:latest
        imagePullPolicy: Always
        name: gitlab-runner
        volumeMounts:
        - mountPath: /etc/gitlab-runner
          name: config
        - mountPath: /etc/ssl/certs
          name: cacerts
          readOnly: true
      restartPolicy: Always
      volumes:
      - configMap:
          name: gitlab-runner
        name: config
      - hostPath:
          path: /usr/share/ca-certificates/mozilla
        name: cacerts
```
