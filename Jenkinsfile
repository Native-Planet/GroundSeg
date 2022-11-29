pipeline {
    agent any
    stages {
        stage('Build') {
            steps {
              sh 'mv ./release/version.csv /opt/groundseg/version/version.csv'
            }
        }
    }
}
