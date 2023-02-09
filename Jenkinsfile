pipeline {
    agent any
    environment {
        tag = sh ( 
            script: '''
                environ=`echo $BRANCH_NAME|sed 's@origin/@@g'`
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
    stages {
        stage('amd64build') {
            steps {
                script {
                    if( "${tag}" == "arm-test" ) {
                        sh '''
                        echo "debug: building amd64"
                        echo mkdir -p /opt/groundseg/version/bin
                        echo cd ./build-scripts
                        echo docker build --tag nativeplanet/groundseg-builder:3.10.9 .
                        echo cd ..
                        echo rm -rf /var/jenkins_home/tmp
                        echo mkdir -p /var/jenkins_home/tmp
                        echo cp -r api /var/jenkins_home/tmp
                        echo docker run -v /home/np/np-cicd/jenkins_conf/tmp/binary:/binary -v /home/np/np-cicd/jenkins_conf/tmp/api:/api nativeplanet/groundseg-builder:3.10.9
                        echo chmod +x /var/jenkins_home/tmp/binary/groundseg
                        echo mv /var/jenkins_home/tmp/binary/groundseg /opt/groundseg/version/bin/groundseg_amd64
                        '''
                    }
                }
            }
        }
        stage('arm64build') {
            agent { node { label 'arm' } }
            steps {
                script {
                    if( "${tag}" == "arm-test" ) {
                        sh '''
                        echo "debug: building arm64"
                        cd build-scripts
                        docker build --tag nativeplanet/groundseg-builder:3.10.9 .
                        cd ..
                        docker run -v "$(pwd)/binary":/binary -v "$(pwd)/api":/api nativeplanet/groundseg-builder:3.10.9
                        cd ui
                        # echo docker buildx build --push --tag nativeplanet/groundseg-webui:${tag} --platform linux/amd64,linux/arm64 .
                        '''
                        stash includes: 'binary/groundseg', name: 'groundseg_arm64'
                    }
                }
            }
                }
        stage('postbuild') {
            steps {
                sh 'echo "debug: post-build actions"'
                dir('/opt/groundseg/version/bin/'){
                unstash 'groundseg_arm64'
                }
                sh 'mv /opt/groundseg/version/bin/groundseg /opt/groundseg/version/bin/groundseg_arm64'
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
