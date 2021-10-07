# sops-converter
This is a fairly simple Kubebuilder project that utilizes Mozilla/Sops.

The goal is to be able to use Sops encryption with Kubernetes Secrets in git.

# Controller
The controller is extremely simple, it decrypts the `data` field of `SopsSecret` objects and inserts it into a `v1/Secret` object with the corresponding `name` and `namespace`.

# Deploy
Raw deployment manifests can be found in `deploy/manifests`  
Kustomize base can be found in `deploy/kustomize/base`

# CLI
CLI binaries can be found attached to github releases.
All functionality is dependant on the `sops` binary being present in `$PATH`.

```
A convenience wrapper for working with SopsSecret objects.

Usage:
  sops-converter [command]

Available Commands:
  completion  generate the autocompletion script for the specified shell
  convert     Converts a kubernetes Secret file to a SopsSecret.
  edit        Opens and decrypts a SopsSecret.
  help        Help about any command

Flags:
  -h, --help   help for sops-converter

Use "sops-converter [command] --help" for more information about a command.
```

## Usage-Example

### Convert
Existing secret manifests can be converted to `SopsSecrets`
```
apiVersion: v1
kind: Secret
type: Opaque
metadata:
  name: test
  namespace: sopssecrets
stringData:
  superSecretPassword: password123
```

NOTE: `data` and `stringData` can both be used.   
Keys in `stringData` will always take priority over `data`.

Args are passed through to `sops --encrypt`.

```
sops-converter convert secret.yaml --kms key:arn:goes:here > output.yaml
```

The output of convert can be applied directly to the cluster.  
`kubectl apply -f output.yaml`
It can then be found in `corev1/Secret` form using:
`kubectl get secret -n sopssecrets test -o yaml`

### Edit

Existing `SopsSecret` manifests can be decrypted and opened into an editor.

```
sops-converter edit output.yaml
```

If you wish to use a different editor such as Atom or Sublime
```
export EDITOR='atom -w'
or
export EDITOR='code -w'
```


### Uninstallation
This controller is safe to uninstall if you follow a few steps first.
This project does not use OwnerReferences as they would make this process much more difficult.

Set the environment variable `DISABLE_FINALIZERS=true`.
Once this finalizer is set, let the controller restart and finish reconciling all objects.
(tail the logs and wait for it to stop)

Once this is done, check your SopsSecret objects. They should no long haver a finalizer set on them.

It is now safe to scale down the controller and delete the CRD. 
All of the secrets created by the controller will remain.


### Prevent deletion of an individual Secret
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


### Template

You can deploy a secret to multiple namespaces or as a different name using the template.

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
This would create a secret in the `example1` and `example2` namespaces and not the `default` namespace.
If you do not specify `spec.template.metadata.namespaces` it will be defaulted to the namespace the SopsSecret object is in.

### IgnoreKeys

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

### Ownership label

This controller uses an ownership label. If the label is not set on a secret the controller will not modify the secret.

The value of the secret is `${Name}.${Namespace}` of the SopsScret object that created it.

```
apiVersion: v1
kind: Secret
metadata:
  name: my-secret
  namespace: default
  labels:
      secrets.dhouti.dev/owned-by-controller: my-secret.default
```