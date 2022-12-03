pipeline {
    agent any
    stages {
        stage('Build') {
            steps {
              sh 'mv ./release/version.csv /opt/groundseg/version/version.csv'
              sh 'mv ./release/version_edge.csv /opt/groundseg/version/version_edge.csv'
            }
        }
    }
}
