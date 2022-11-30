pipeline {
    agent any
    stages {
        stage('Build') {
            steps {
              sh 'mv ./release/version.csv /opt/groundseg/version/version.csv'
              sh 'mv ./release/version_staging.csv /opt/groundseg/version/version_staging.csv'
            }
        }
    }
}
