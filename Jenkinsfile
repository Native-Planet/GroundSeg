pipeline {
    agent any
    parameters {
        gitParameter name: 'RELEASE_TAG',
                     type: 'PT_BRANCH_TAG',
                     defaultValue: 'master'
    }
    environment {
        channel = sh ( 
            script: '''
                environ=`echo $BRANCH_NAME|sed 's@origin/@@g'`
                if [ "${environ}" = "master" ]; then
                    echo "latest"
                elif [ "${environ}" = "dev" ]; then
                    echo "edge"
                else
                    echo "nobuild"
                fi
            ''',
            returnStdout: true
        ).trim()
        versionauth = credentials('VersionAuth')
        tag = "${params.RELEASE_TAG}"
    }
    stages {
        stage('checkout') {
            steps {
                checkout([$class: 'GitSCM',
                          branches: [[name: "${params.RELEASE_TAG}"]],
                          doGenerateSubmoduleConfigurations: false,
                          extensions: [],
                          gitTool: 'Default',
                          submoduleCfg: [],
                          userRemoteConfigs: [[credentialsId: 'Github token', url: 'https://github.com/Native-Planet/GroundSeg.git']]
                        ])
            }
        }/*
        stage('amd64 build') {
            steps {
                script {
                    if( "${channel}" != "nobuild" ) {
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
                            mv /var/jenkins_home/tmp/binary/groundseg /opt/groundseg/version/bin/groundseg_amd64_${tag}
                        '''
                    }
                }
            }
        }
        stage('arm64 build') {
            agent { node { label 'arm' } }
            steps {
                checkout([$class: 'GitSCM',
                          branches: [[name: "${params.RELEASE_TAG}"]],
                          doGenerateSubmoduleConfigurations: false,
                          extensions: [],
                          gitTool: 'Default',
                          submoduleCfg: [],
                          userRemoteConfigs: [[credentialsId: 'Github token', url: 'https://github.com/Native-Planet/GroundSeg.git']]
                        ])
                script {
                    if( "${channel}" != "nobuild" ) {
                        sh '''
                            echo "debug: building arm64"
                            cd build-scripts
                            docker build --tag nativeplanet/groundseg-builder:3.10.9 .
                            cd ..
                            docker run -v "$(pwd)/binary":/binary -v "$(pwd)/api":/api nativeplanet/groundseg-builder:3.10.9
                            cd ui
                            docker buildx build --push --tag nativeplanet/groundseg-webui:${channel} --platform linux/amd64,linux/arm64 .
                        '''
                        stash includes: 'binary/groundseg', name: 'groundseg_arm64'
                        sh '''
                            environ=`echo $BRANCH_NAME|sed 's@origin/@@g'`
                            cd ..
                            rm -rf GroundSeg_${environ}
                        '''
                    }
                }
            }
        }
        stage('move binaries') {
            steps {
                script {
                    if( "${channel}" != "nobuild" ){  
                        sh 'echo "debug: post-build actions"'
                        dir('/opt/groundseg/version/bin/'){
                        unstash 'groundseg_arm64'
                        }
                        sh 'mv /opt/groundseg/version/bin/binary/groundseg /opt/groundseg/version/bin/groundseg_arm64_${tag}'
                        sh 'rm -rf /opt/groundseg/version/bin/binary/'
                    }
                }
            }
        } */
        stage('version update') {
            environment {
                armsha = sh(
                    script: '''#!/bin/bash -x
                        val=`sha256sum /opt/groundseg/version/bin/groundseg_arm64_${tag}|awk '{print \$1}'`
                        echo ${val}
                    ''',
                    returnStdout: true
                ).trim()
                amdsha = sh(
                    script: '''#!/bin/bash -x
                        val=`sha256sum /opt/groundseg/version/bin/groundseg_amd64_${tag}|awk '{print \$1}'`
                        echo ${val}
                    ''',
                    returnStdout: true
                ).trim()
                dockerhash = sh(
                    script: '''#!/bin/bash -x
                        obj=`curl -s "https://hub.docker.com/v2/repositories/nativeplanet/groundseg-webui/tags/${channel}/?page_size=100" | jq -r '.digest'`
                        echo $obj|sed 's/sha256://g'
                    ''',
                    returnStdout: true
                ).trim()
                major = sh(
                    script: '''#!/bin/bash -x
                        ver=${tag}
                        if [[ "${tag}" == *"-"* ]]; then
                            ver=`echo ${tag}|awk -F '-' '{print \$2}'`
                        fi
                        major=`echo ${ver}|awk -F '.' '{print \$1}'|sed 's/v//g'`
                        echo ${major}
                    ''',
                    returnStdout: true
                ).trim()
                minor = sh(
                    script: '''#!/bin/bash -x
                        ver=${tag}
                        if [[ "${tag}" == *"-"* ]]; then
                            ver=`echo ${tag}|awk -F '-' '{print \$2}'`
                        fi
                        minor=`echo ${ver}|awk -F '.' '{print \$2}'|sed 's/v//g'`
                        echo ${minor}
                    ''',
                    returnStdout: true
                ).trim()
                patch = sh(
                    script: '''#!/bin/bash -x
                        ver=${tag}
                        if [[ "${tag}" == *"-"* ]]; then
                            ver=`echo ${tag}|awk -F '-' '{print \$2}'`
                        fi
                        patch=`echo ${ver}|awk -F '.' '{print \$3}'|sed 's/v//g'`
                        echo ${patch}
                    ''',
                    returnStdout: true
                ).trim()
                armbin = "https://bin.infra.native.computer/groundseg_arm64_${tag}"
                amdbin = "https://bin.infra.native.computer/groundseg_amd64_${tag}"
            }
            steps {
                script {
                    if( "${channel}" == "latest" ) {
                        sh '''#!/bin/bash -x
                            mv ./release/standard_install.sh /opt/groundseg/get/install.sh
                            mv ./release/groundseg_install.sh /opt/groundseg/get/only.sh
                            curl -X PUT -H "X-Api-Key: ${versionauth}" -H 'Content-Type: application/json' \
                                https://version.groundseg.app/groundseg/latest/groundseg/amd64_url/payload \
                                -d "{\"value\":\"${amdbin}\"}"
                            curl -X PUT -H "X-Api-Key: ${versionauth}" -H 'Content-Type: application/json' \
                                https://version.groundseg.app/groundseg/latest/groundseg/arm64_url/payload \
                                -d "{\"value\":\"${armbin}\"}"
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://version.groundseg.app/groundseg/latest/groundseg/amd64_sha256/${amdsha}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://version.groundseg.app/groundseg/latest/groundseg/arm64_sha256/${armsha}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://version.groundseg.app/groundseg/latest/webui/sha256/${dockerhash}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://version.groundseg.app/groundseg/latest/groundseg/major/${major}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://version.groundseg.app/groundseg/latest/groundseg/minor/${minor}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://version.groundseg.app/groundseg/latest/groundseg/patch/${patch}
                        '''
                    }
                    if( "${channel}" == "edge" ) {
                        sh '''#!/bin/bash -x
                            curl -X PUT -H "X-Api-Key: ${versionauth}" -H 'Content-Type: application/json' \
                                https://version.groundseg.app/groundseg/edge/groundseg/amd64_url/payload \
                                -d "{\"value\":\"${amdbin}\"}"
                            curl -X PUT -H "X-Api-Key: ${versionauth}" -H 'Content-Type: application/json' \
                                https://version.groundseg.app/groundseg/edge/groundseg/arm64_url/payload \
                                -d "{\"value\":\"${armbin}\"}"
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://version.groundseg.app/groundseg/edge/groundseg/amd64_sha256/${amdsha}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://version.groundseg.app/groundseg/edge/groundseg/arm64_sha256/${armsha}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://version.groundseg.app/groundseg/edge/webui/sha256/${dockerhash}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://version.groundseg.app/groundseg/edge/groundseg/major/${major}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://version.groundseg.app/groundseg/edge/groundseg/minor/${minor}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://version.groundseg.app/groundseg/edge/groundseg/patch/${patch}
                        '''
                    }
                }
            }
        }
        stage('merge to master') {
            steps {
                script {
                    if( "${channel}" == "latest" ) {
                        sh (
                            script: '''
                                git checkout -b tag-${tag}
                                git checkout master
                                git merge tag-${tag}
                                git commit -am "merged ${tag} release"
                                git push origin master
                            '''
                        )
                    }
                }
            }
        }
    }
        post {
            always {
                cleanWs(cleanWhenNotBuilt: true,
                    deleteDirs: true,
                    disableDeferredWipeout: false,
                    notFailBuild: true,
                    patterns: [[pattern: '.gitignore', type: 'INCLUDE'],
                               [pattern: '.propsfile', type: 'EXCLUDE']])
            }
        }
}
