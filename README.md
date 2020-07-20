# sops-converter
This is a very simple Kubebuilder project that heavily utilizes Mozilla/Sops.

The goal is to be able to use Sops encryption with Kubernetes Secrets in git.

# Controller
The controller is extremely simple, it decrypts the `data` field of `SopsSecret` objects and inserts it into a `corev1/Secret` object with the corresponding `name` and `namespace`.

# Deploy
Raw deployment manifests can be found in `deploy/manifests`  
Kustomize base can be found in `deploy/kustomize/base`

# CLI
Binaries are not packaged and released for now.
You can clone the repo and `make build`  
The output can be found in `bin/sops-converter`

```
A convenience wrapper for working with SopsSecret objects.

Usage:
  sops-converter [command]

Available Commands:
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
export EDITOR='subl -w'
```
