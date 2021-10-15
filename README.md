# sops-converter
The goal of this project is to be able to use Sops encryption with Kubernetes Secrets so they can be stored safely in Git.


# Controller
The controller is fairly simple, it decrypts the `data` field of `SopsSecret` objects and inserts it into a `v1/Secret` object with the corresponding `name` and `namespace`.


# CLI
There is a helper CLI to convert existing Secrets to SopsSecrets.
The CLI can also be used to edit secrets from their encrypted form.

Examples of usage can be found in `docs/cli/README.md`


# Deploy
Kustomize base can be found in `deploy/kustomize/base`
Examples of deployments with Kustomize can be found in `docs/examples`


## Uninstallation
This controller is safe to uninstall if you follow a few steps first.

Set the environment variable `DISABLE_FINALIZERS=true`.
Once this finalizer is set, let the controller restart and finish reconciling all objects.
(tail the logs and wait for it to stop)

Once this is done, check your SopsSecret objects. They should no long haver a finalizer set on them.

It is now safe to scale down the controller and delete the CRD. 
All of the secrets created by the controller will remain.

There was a concious decision to not use OwnerReferences as they would make this process more difficult and are in general less safe for something as critical as Secrets.


## Ownership label

This controller uses an ownership label. If the label is not set on a secret the controller will not modify the secret.
This kind of label is common in other secret operators, but was also a requirement here because of the lack of OwnerReferences.

```
apiVersion: v1
kind: Secret
metadata:
  name: my-secret
  namespace: default
  labels:
      secrets.dhouti.dev/owned-by-controller: my-secret.default
```
The value of the label is `${Name}.${Namespace}` of the SopsScret object that created it.


## Prevent deletion of an individual Secret
If you wish to delete a SopsSecret object and have the Secret remain you can set skipFinalizers.
```
apiVersion: secrets.dhouti.dev/v1beta1
kind: SopsSecret
metadata:
  name: my-secret
  namespace: default
spec:
  skipFinalizers: true
```

Once this value is set you can delete the SopsSecret object and the underlying secret will remain.


## Template
You can deploy a secret to different namespaces or as a different name using the template.

Usage:
```
apiVersion: secrets.dhouti.dev/v1beta1
kind: SopsSecret
metadata:
  name: my-secret
  namespace: default
spec:
  template:
    metadata:
      name: new-secret
      namespaces:
      - example1
      - example2
```
This would create a secret named `new-secret` in the `example1` and `example2` namespaces and not the `default` namespace.
If you do not specify `spec.template.metadata.namespaces` it will be defaulted to the namespace the SopsSecret object is in.
If you do not specify `.spec.template.metadata.name` it will be defaulted to the name of the SopsSecret object.


## IgnoreKeys

You can prevent the controller from managing keys in the output secret.
One use case would be ArgoCD. ArgoCD creates some keys in secret objects on controller start, you can let argocd create them instead of specifying them yourself.
```
apiVersion: secrets.dhouti.dev/v1beta1
kind: SopsSecret
metadata:
  name: argocd-secret
  namespace: argocd
type: Opaque
spec:
  ignoredKeys:
  - tls.crt
  - tls.key
  - server.secretKey
```