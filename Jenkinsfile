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
                        docker build --tag nativeplanet/groundseg-builder:3.10.9 .
                        cd .. && docker run -v "$(pwd)/binary":/binary -v "$(pwd)/api":/api nativeplanet/groundseg-builder:3.10.9
                        chmod +x ./binary/groundseg
                        mv ./binary/groundseg /opt/groundseg/version/bin/groundseg_x64
                        '''
                    }
                }
            }
        }
        node('NP ARM server') {
            stages {
                stage('arm64build') {
                    steps {
                        git url: 'https://github.com/Native-Planet/GroundSeg.git'
                        script {
                            if( "${tag}" == "latest" ) {
                                sh '''
                                cd build-scripts
                                docker build --tag nativeplanet/groundseg-builder:3.10.9 .
                                cd ..
                                docker run -v "$(pwd)/binary":/binary -v "$(pwd)/api":/api nativeplanet/groundseg-builder:3.10.9
                                mv binary/groundseg binary/groundseg_arm64
                                cd ui
                                docker buildx build --push --tag nativeplanet/groundseg-webui:latest --platform linux/amd64,linux/arm64 .
                                '''
                            }
                        }
                    }
                }
                stage('stash') {
                    stash includes: 'binary/**', name: 'groundseg_arm64'
                }
            }
        }
    stages {
        stage('unstash') {
            String binPath = '/opt/groundseg/version/bin/'
            dir (binPath) {
                unstash 'groundseg_arm64'
            }
        }
        stage('postbuild') {
            steps {
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
    post {
        always {
            cleanWs deleteDirs: true, notFailBuild: true
        }
    }
}
