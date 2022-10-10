pipeline {

  agent {label 'ubuntu'}

  environment {
    module = "plugin-jenkins-master"
    version = "qa"
    repo = "853385135017.dkr.ecr.us-east-1.amazonaws.com/compliance-hub/$module"
    target = "$repo:$version"
    USER = "ansible-db"
    TOKEN = credentials('GitHub-ansible-db')
  }

  options { disableConcurrentBuilds() }

  stages {

    stage('Building image') {
      steps{
        withCredentials([string(credentialsId: 'AWS_ACCESS_KEY_ID', variable: 'AWS_ACCESS_KEY_ID'), string(credentialsId: 'AWS_SECRET_ACCESS_KEY', variable: 'AWS_SECRET_ACCESS_KEY')]) {
          script {
            currentBuild.description = "${env.GIT_BRANCH} ${env.GIT_COMMIT}"
            sh "docker build -t $target --build-arg=USER --build-arg=TOKEN ."
          }
        }
      }
    }

    stage('Pushing image') {
      steps{
        withCredentials([string(credentialsId: 'AWS_ACCESS_KEY_ID', variable: 'AWS_ACCESS_KEY_ID'), string(credentialsId: 'AWS_SECRET_ACCESS_KEY', variable: 'AWS_SECRET_ACCESS_KEY')]) {
          script {
            sh "docker push $target"
          }
        }
      }
    }

    stage('Deploy') {
      steps {
        build job: 'compliance-hub-service-deploy', parameters: [string(name: 'service',  value: "${module}"), string(name: 'version',  value: "qa"), string(name: 'platform',  value: "qa")]
      }
    }

  }
}

