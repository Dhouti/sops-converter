pipeline {
  triggers {
      githubPush()
  }
  agent {
    kubernetes {
    }
  }
  stages {
    podTemplate(containers: [
        containerTemplate(name: 'sops-converter-builder', image: 'docker.dhouti.dev/sops-converter-builder:v0.0.1', ttyEnabled: true, command: 'cat'),
        containerTemplate(name: 'kaniko', image: 'gcr.io/kaniko-project/executor:v1.3.0', ttyEnabled: true, command: 'cat')
    ]) {
      node(POD_LABEL) {
        stage('Run tests') {
          steps {
            container(name: 'sops-converter-builder', shell: '/bin/bash') {
              sh '''
                go test ./... -coverprofile cover.out
              '''
            }
          }
        }
        stage('Build') {
          steps {
            container(name: 'kaniko', shell: '/bin/bash') {
              sh '''
                /kaniko/executor --dockerfile . --destination docker.dhouti.dev/sops-converter:jenkins-test
              '''
            }
          }
        }
      }
    }
  }
}
