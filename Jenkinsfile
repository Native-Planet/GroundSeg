pipeline {
    agent any
    parameters {
        gitParameter(
            name: 'RELEASE_TAG',
            type: 'PT_BRANCH_TAG',
            defaultValue: 'master')
        choice(
            choices: ['no' , 'yes'],
            description: 'Merge tag into master branch (doesn\'t do anything in dev)',
            name: 'MERGE')
    }
    environment {
        /* translate git branch to release channel */
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
        /* version server auth header */
        versionauth = credentials('VersionAuth')
        /* release tag to be built*/
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
        } /*
        stage('SonarQube') {
            environment {
                scannerHome = "${tool 'SonarQubeScanner'}"
            }
            steps {
                withSonarQubeEnv('SonarQube') {
                    sh "${scannerHome}/bin/sonar-scanner -Dsonar.projectKey=Native-Planet_GroundSeg_AYZoKNgHuu12TOn3FQ6N -Dsonar.python.version=3.11"
                }
            }
        } */
        stage('build') {
            stage('amd64 build') {
                steps {
                    /* build binaries and move to web dir */
                    script {
                        if(( "${channel}" != "nobuild" ) && ( "${channel}" != "latest" )) {
                            sh '''
                                git checkout ${tag}
                                mkdir -p /opt/groundseg/version/bin
                                cd ./goseg
                                env GOOS=linux GOARCH=amd64 go build -o /opt/groundseg/version/bin/groundseg_amd64_${tag}_${channel}
                                env GOOS=linux GOARCH=arm64 go build -o /opt/groundseg/version/bin/groundseg_arm64_${tag}_${channel}
                            '''
                        }
                        if( "${channel}" == "latest" ) {
                            sh '''
                                cp /opt/groundseg/version/bin/groundseg_amd64_${tag}_edge /opt/groundseg/version/bin/groundseg_amd64_${tag}_${channel}
                                cp /opt/groundseg/version/bin/groundseg_arm64_${tag}_edge /opt/groundseg/version/bin/groundseg_arm64_${tag}_${channel}
                            '''
                        }
                    }
                }
            }
        }
        stage('move binaries') {
            steps {
                /* unstash arm binary on master server */
                script {
                    if(( "${channel}" != "nobuild" ) && ( "${channel}" != "latest" )) {  
                        sh 'echo "debug: post-build actions"'
                        sh '''#!/bin/bash -x
                        rclone -vvv --config /var/jenkins_home/rclone.conf copy /opt/groundseg/version/bin/groundseg_arm64_${tag}_${channel} r2:groundseg/bin
                        rclone -vvv --config /var/jenkins_home/rclone.conf copy /opt/groundseg/version/bin/groundseg_amd64_${tag}_${channel} r2:groundseg/bin
                        '''
                    }
                }
            }
        }
        stage('version update') {
            environment {
                /* update versions and hashes on public version server */
                armsha = sh(
                    script: '''#!/bin/bash -x
                        val=`sha256sum /opt/groundseg/version/bin/groundseg_arm64_${tag}_${channel}|awk '{print \$1}'`
                        echo ${val}
                    ''',
                    returnStdout: true
                ).trim()
                amdsha = sh(
                    script: '''#!/bin/bash -x
                        val=`sha256sum /opt/groundseg/version/bin/groundseg_amd64_${tag}_${channel}|awk '{print \$1}'`
                        echo ${val}
                    ''',
                    returnStdout: true
                ).trim()
                webui_amd64_hash = sh(
                    script: '''#!/bin/bash -x
                    curl -s "https://hub.docker.com/v2/repositories/nativeplanet/groundseg-webui/tags/${channel}/?page_size=100" \
                    |jq -r '.images[]|select(.architecture=="amd64").digest'|sed 's/sha256://g'
                    ''',
                    returnStdout: true
                ).trim()
                webui_arm64_hash = sh(
                    script: '''#!/bin/bash -x
                    curl -s "https://hub.docker.com/v2/repositories/nativeplanet/groundseg-webui/tags/${channel}/?page_size=100" \
                    |jq -r '.images[]|select(.architecture=="arm64").digest'|sed 's/sha256://g'
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
                armbin = "https://files.native.computer/bin/groundseg_arm64_${tag}_${channel}"
                amdbin = "https://files.native.computer/bin/groundseg_amd64_${tag}_${channel}"
            }
            steps {
                script {
                    if( "${channel}" == "latest" ) {
                        sh '''#!/bin/bash -x
                            mv ./release/standard_install.sh /opt/groundseg/get/install.sh
                            mv ./release/groundseg_install.sh /opt/groundseg/get/only.sh
                            webui_amd64_hash=`curl https://version.groundseg.app | jq -r '.[].edge.webui.amd64_sha256'`
                            webui_arm64_hash=`curl https://version.groundseg.app | jq -r '.[].edge.webui.arm64_sha256'`
                            curl -X PUT -H "X-Api-Key: ${versionauth}" -H 'Content-Type: application/json' \
                                https://version.groundseg.app/modify/groundseg/latest/groundseg/amd64_url/payload \
                                -d "{\\"value\\":\\"${amdbin}\\"}"
                            curl -X PUT -H "X-Api-Key: ${versionauth}" -H 'Content-Type: application/json' \
                                https://version.groundseg.app/modify/groundseg/latest/groundseg/arm64_url/payload \
                                -d "{\\"value\\":\\"${armbin}\\"}"
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://version.groundseg.app/modify/groundseg/latest/groundseg/amd64_sha256/${amdsha}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://version.groundseg.app/modify/groundseg/latest/groundseg/arm64_sha256/${armsha}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://version.groundseg.app/modify/groundseg/latest/webui/amd64_sha256/${webui_amd64_hash}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://version.groundseg.app/modify/groundseg/latest/webui/arm64_sha256/${webui_arm64_hash}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://version.groundseg.app/modify/groundseg/latest/groundseg/major/${major}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://version.groundseg.app/modify/groundseg/latest/groundseg/minor/${minor}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://version.groundseg.app/modify/groundseg/latest/groundseg/patch/${patch}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" -H 'Content-Type: application/json' \
                                https://version.groundseg.app/modify/groundseg/canary/groundseg/amd64_url/payload \
                                -d "{\\"value\\":\\"${amdbin}\\"}"
                            curl -X PUT -H "X-Api-Key: ${versionauth}" -H 'Content-Type: application/json' \
                                https://version.groundseg.app/modify/groundseg/canary/groundseg/arm64_url/payload \
                                -d "{\\"value\\":\\"${armbin}\\"}"
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://version.groundseg.app/modify/groundseg/canary/groundseg/amd64_sha256/${amdsha}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://version.groundseg.app/modify/groundseg/canary/groundseg/arm64_sha256/${armsha}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://version.groundseg.app/modify/groundseg/canary/webui/amd64_sha256/${webui_amd64_hash}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://version.groundseg.app/modify/groundseg/canary/webui/arm64_sha256/${webui_arm64_hash}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://version.groundseg.app/modify/groundseg/canary/groundseg/major/${major}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://version.groundseg.app/modify/groundseg/canary/groundseg/minor/${minor}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://version.groundseg.app/modify/groundseg/canary/groundseg/patch/${patch}
                        '''
                    }
                    if( "${channel}" == "edge" ) {
                        sh '''#!/bin/bash -x
                            curl -X PUT -H "X-Api-Key: ${versionauth}" -H 'Content-Type: application/json' \
                                https://version.groundseg.app/modify/groundseg/edge/groundseg/amd64_url/payload \
                                -d "{\\"value\\":\\"${amdbin}\\"}"
                            curl -X PUT -H "X-Api-Key: ${versionauth}" -H 'Content-Type: application/json' \
                                https://version.groundseg.app/modify/groundseg/edge/groundseg/arm64_url/payload \
                                -d "{\\"value\\":\\"${armbin}\\"}"
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://version.groundseg.app/modify/groundseg/edge/groundseg/amd64_sha256/${amdsha}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://version.groundseg.app/modify/groundseg/edge/groundseg/arm64_sha256/${armsha}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://version.groundseg.app/modify/groundseg/edge/webui/amd64_sha256/${webui_amd64_hash}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://version.groundseg.app/modify/groundseg/edge/webui/arm64_sha256/${webui_arm64_hash}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://version.groundseg.app/modify/groundseg/edge/groundseg/major/${major}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://version.groundseg.app/modify/groundseg/edge/groundseg/minor/${minor}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://version.groundseg.app/modify/groundseg/edge/groundseg/patch/${patch}
                        '''
                    }
                }
            }
        }
        stage('merge to master') {
            steps {
                /* merge tag changes into master if deploying to master */
                script {
                    if(( "${channel}" == "latest" ) && ( "${params.MERGE}" == "yes" )) {
                        withCredentials([gitUsernamePassword(credentialsId: 'Github token', gitToolName: 'Default')]) {
			    sh (
                                script: '''
                                    git checkout master
                                    git merge ${tag} -m "Merged ${tag}"
                                    git push
                                '''
                            )
			}
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
