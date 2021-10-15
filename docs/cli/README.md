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

## Convert
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

### Labels/Annotations
Any labels/Annotations that are present on the secret will be added to the SopsSecret in the template.metadata.labels/annotations fields.  


## Edit

Existing `SopsSecret` manifests can be decrypted and opened into an editor.
This supports multiple kubernetes documents in a single yaml file, ie:
```
key: value
---
key: value
---
```
If there are SopsSecret objects present you will be prompted which you would like to edit.


If you wish to use a different editor such as VSCode or Atom
```
export EDITOR='code -w'
or
export EDITOR='atom -w'
```


### Usage
```
sops-converter edit output.yaml
```