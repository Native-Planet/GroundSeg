pipeline {
    agent any
    environment {
        environ = sh ( 
            script: '''
                echo $BRANCH_NAME|sed 's@origin/@@g'
            ''',
            returnStdout: true
        ).trim()
    }
    stages {
        stage('Build') {
            environment {
                tag = sh ( 
                    script: '''
                        if [ "${environ}" = "main" ]; then
                            echo "latest"
                        elif [ "${environ}" = "edge" ]; then
                            echo "edge"
                        else
                            echo "nobuild"
                        fi
                    ''',
                    returnStdout: true
                ).trim()
            }
            steps {
                script {
                    if( "${tag}" == "latest" ) {
                        sh 'mv ./release/version.csv /opt/groundseg/version/'
                        sh 'mv ./release/standard_install.sh /opt/groundseg/get/install.sh'
                        sh 'mv ./release/groundseg_install.sh /opt/groundseg/get/only.sh'
                    }
                    if( "${tag}" == "edge" ) {
                        sh 'mv ./release/version_edge.csv /opt/groundseg/version/'
                    }
                }
            }
        }
    }
}
