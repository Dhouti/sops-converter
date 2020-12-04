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
  - name: kaniko
    image: gcr.io/kaniko-project/executor:v1.3.0-debug
    command:
    - /busybox/cat
    tty: true
"""
    }
  }

  stages {
    stage('Run tests') {
      steps {
        container(name: 'sops-converter-builder', shell: '/bin/bash') {
        sh '''
            export PATH=$PATH:/usr/local/kubebuilder/bin
            make test
          '''
        }
      }
    }
    stage('Build') {
      steps {
        container(name: 'kaniko', shell: '/busybox/sh') {
          sh '''
            /kaniko/executor --dockerfile . --destination docker.dhouti.dev/sops-converter:jenkins-test
          '''
        }
      }
    }
  }
}
