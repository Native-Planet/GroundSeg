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
        stage('amd64build') {
            environment {
                tag = sh ( 
                    script: '''
                        if [ "${environ}" = "main" ]; then
                            echo "latest"
                        elif [ "${environ}" = "edge" ]; then
                            echo "edge"
                        elif [ "${environ}" = "arm-test" ]; then
                            echo "arm-test"
                        else
                            echo "nobuild"
                        fi
                    ''',
                    returnStdout: true
                ).trim()
            }
            steps {
                script {
                    if( "${tag}" == "arm-test" ) {
                        sh '''
                        echo "debug: building amd64"
                        mkdir -p /opt/groundseg/version/bin
                        cd ./build-scripts
                        docker build --tag nativeplanet/groundseg-builder:3.10.9 .
                        cd ..
                        rm -rf /var/jenkins_home/tmp
                        mkdir -p /var/jenkins_home/tmp
                        cp -r api /var/jenkins_home/tmp
                        docker run -v /home/np/np-cicd/jenkins_conf/tmp/binary:/binary -v /home/np/np-cicd/jenkins_conf/tmp/api:/api nativeplanet/groundseg-builder:3.10.9
                        chmod +x /var/jenkins_home/tmp/binary/groundseg
                        mv /var/jenkins_home/tmp/binary/groundseg /opt/groundseg/version/bin/groundseg_amd64
                        '''
                    }
                }
            }
        }
        stage('arm64build') {
            environment {
                tag = sh ( 
                    script: '''
                        if [ "${environ}" = "main" ]; then
                            echo "latest"
                        elif [ "${environ}" = "edge" ]; then
                            echo "edge"
                        elif [ "${environ}" = "arm-test" ]; then
                            echo "arm-test"
                        else
                            echo "nobuild"
                        fi
                    ''',
                    returnStdout: true
                ).trim()
            }
            agent { node { label 'arm' } }
            steps {
                script {
                    if( "${tag}" == "arm-test" ) {
                        sh '''
                        cd build-scripts
                        docker build --tag nativeplanet/groundseg-builder:3.10.9 .
                        cd ..
                        docker run -v "$(pwd)/binary":/binary -v "$(pwd)/api":/api nativeplanet/groundseg-builder:3.10.9
                        mv binary/groundseg binary/groundseg_arm64
                        cd ui
                        # echo docker buildx build --push --tag nativeplanet/groundseg-webui:latest --platform linux/amd64,linux/arm64 .
                        '''
                        stash includes: 'binary/groundseg_arm64', name: 'groundseg_arm64'
                    }
                }
            }
                }
        stage('postbuild') {
            environment {
                tag = sh ( 
                    script: '''
                        if [ "${environ}" = "main" ]; then
                            echo "latest"
                        elif [ "${environ}" = "edge" ]; then
                            echo "edge"
                        elif [ "${environ}" = "arm-test" ]; then
                            echo "arm-test"
                        else
                            echo "nobuild"
                        fi
                    ''',
                    returnStdout: true
                ).trim()
            }
            steps {
                dir('/opt/groundseg/version/bin/'){
                unstash 'groundseg_arm64'
                }
                script {
                    if( "${tag}" == "latest" ) {
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
