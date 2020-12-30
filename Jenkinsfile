pipeline {
  triggers {
      githubPush()
  }
  agent {
    kubernetes {
    yaml """
apiVersion: v1
kind: Pod
spec:
  containers:
  - name: sops-converter-builder
    image: docker.dhouti.dev/sops-converter-builder:v0.0.1
    command:
    - cat
    tty: true
    resources:
      requests:
        cpu: 1
        memory: 750Mi
  - name: dind
    image: docker.dhouti.dev/dind-buildx:v0.0.1
    tty: true
    securityContext:
      privileged: true
"""
    }
  }

  stages {
    //stage('Run tests') {
    //  steps {
    //    container(name: 'sops-converter-builder', shell: '/bin/bash') {
    //    sh '''
    //        make test
    //      '''
    //    }
    //  }
    //}
    stage('Build Master') {
      steps {
        container(name: 'dind', shell: '/bin/sh') {
        sh '''
            docker buildx create --use
            docker buildx build --platform linux/amd64,linux/arm64 -t docker.dhouti.dev/sops-converter:${GIT_COMMIT:0:7} . --push
          '''
        }
      }
    }
  }
}
