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
                        sh '''
                        mkdir -p /opt/groundseg/version/bin && cd ./build-scripts
                        docker build --tag nativeplanet/groundseg-builder:3.10.9-test .
                        cd .. && docker run -v "$(pwd)/binary":/binary -v "$(pwd)/api":/api nativeplanet/groundseg-builder:3.10.9
                        chmod +x ./binary/groundseg
                        mv ./binary/groundseg /opt/groundseg/version/bin/groundseg_x64
                        '''
                        build job: "GroundSeg-ARM", wait: true
                        sh '''
                        mv ./release/version.csv /opt/groundseg/version/
                        mv ./release/standard_install.sh /opt/groundseg/get/install.sh
                        mv ./release/groundseg_install.sh /opt/groundseg/get/only.sh
                        '''
                    }
                    if( "${tag}" == "edge" ) {
                        sh 'mv ./release/version_edge.csv /opt/groundseg/version/'
                    }
                }
            }
        }
    }
    post {
        always {
            cleanWs deleteDirs: true, notFailBuild: true
        }
    }
}
