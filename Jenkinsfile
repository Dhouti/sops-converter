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
            make test
          '''
        }
      }
    }
    stage('Build Master') {
      when {
        branch 'master'
      }
      steps {
        container(name: 'kaniko', shell: '/busybox/sh') {
          sh '''
            /kaniko/executor --context "dir:///$(pwd)" --destination docker.dhouti.dev/sops-converter:${GIT_COMMIT:0:7}
          '''
        }
      }
    }

    stage('Build Release Tag') {
      when {
        buildingTag()
      }
      container(name: 'kaniko', shell: '/busybox/sh') {
        sh '''
          /kaniko/executor --context "dir:///$(pwd)" --destination docker.dhouti.dev/sops-converter:${TAG_NAME}
        '''
      }
    }
  }
}
