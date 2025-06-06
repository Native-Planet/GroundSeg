pipeline {
    agent any
    parameters {
        gitParameter(
            name: 'RELEASE_TAG',
            type: 'PT_BRANCH_TAG',
            defaultValue: 'master')
        choice(
            choices: ['nobuild', 'edge', 'canary', 'latest'],
            description: 'Publish to release channel',
            name: 'CHANNEL'
        )
        booleanParam(
            name: 'TO_CANARY',
            defaultValue: false,
            description: 'Also push build to canary channel (if edge)'
        )
        choice(
            choices: ['build','promote'],
            description: 'Build a release candidate tag for edge or promote an existing RC to latest (only works with `v2.X.X-rcX` tags)',
            name: 'PROMOTE'
        )
        choice(
            choices: ['staging.version.groundseg.app' , 'version.groundseg.app'],
            description: 'Choose version server',
            name: 'VERSION_SERVER'
        )
    }
    environment {
        /* choose release channel based on params */
        PROMOTE = "${params.PROMOTE}"
        channel = "${params.CHANNEL}"
        /* version server auth header */
        versionauth = credentials('VersionAuth')
        npGhToken = credentials('NPJenkinsGH')
        /* release tag to be built*/
        tag = "${params.RELEASE_TAG}"
        /* staging or production version server */
        version_server = "${params.VERSION_SERVER}"
        to_canary = "${params.TO_CANARY}"
    }
    stages {
        stage('determine channel') {
            steps {
                script {
                    def channelValue = sh(
                        script: """#!/bin/bash -x
                            echo "${params.CHANNEL}"
                        """, 
                        returnStdout: true
                    ).trim()
                    echo "Channel: ${channelValue}"
                    def binTagValue = ''
                    if (channelValue == "latest") {
                        binTagValue = env.tag.tokenize('-')[0]
                    } else {
                        binTagValue = env.tag
                    }
                    echo "BinTag: ${binTagValue}"
                    env.channel = channelValue
                    env.binTag = binTagValue
                }
            }
        }
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
        }
        stage('build') {
            steps {
                /* build binaries and move to web dir */
                script {
                    if(( "${params.CHANNEL}" != "nobuild" ) && ( "${params.CHANNEL}" != "latest" )) {
                        /* Goseg */
                        sh '''#!/bin/bash -x
                            git checkout ${tag}
                            cd ./ui
                            DOCKER_BUILDKIT=0 docker build -t web-builder -f builder.Dockerfile .
                            container_id=$(docker create web-builder)
                            docker cp $container_id:/webui/build ./web
                            rm -rf ../goseg/web
                            mv web ../goseg/
                        '''
                        /* Gallseg */
                        sh '''#!/bin/bash -x
                            cd ./ui
                            DOCKER_BUILDKIT=0 docker build -t web-builder -f gallseg.Dockerfile .
                            container_id=$(docker create web-builder)
                            git clone https://github.com/Native-Planet/globber
                            cd globber
                            docker cp $container_id:/webui/build ./web
                            ./glob.sh web
                            hash=$(ls -1 -c . | head -1 | sed "s/glob-\\([a-z0-9\\.]*\\).glob/\\1/")
                            globurl="https://files.native.computer/glob/gallseg-${tag}-${hash}.glob"
                            echo "hash=${hash}" > /opt/groundseg/version/glob/globhash.env
                            echo ${globurl} > /opt/groundseg/version/glob/globurl.txt
                            mv ./*.glob /opt/groundseg/version/glob/gallseg-${tag}-${hash}.glob
                            cd ..
                            rm -rf globber
                            cd ..
                            docketinfo="    glob-http+['${globurl}' ${hash}]"
                            sed -i "/glob-http/c\${docketinfo}" gallseg/desk.docket-0
                            echo "~lablet-nallux-dozryl" > gallseg/desk.ship
                            rm -r /opt/groundseg/release-ships/edge/groundseg
                            cp -r gallseg /opt/groundseg/release-ships/edge/groundseg
                            /opt/groundseg/release-ships/click/click -k -i /opt/groundseg/release-ships/commit.hoon /opt/groundseg/release-ships/edge
                        '''
                        sh """#!/bin/bash -x
                            cd ./goseg
                            env GOOS=linux CGO_ENABLED=0 GOARCH=amd64 go build -o /opt/groundseg/version/bin/groundseg_amd64_${env.binTag}_${env.channel}
                            env GOOS=linux CGO_ENABLED=0 GOARCH=arm64 go build -o /opt/groundseg/version/bin/groundseg_arm64_${env.binTag}_${env.channel}
                        """
                    }
                    /* production releases get promoted from edge */
                    if( "${params.CHANNEL}" == "latest" ) {
                        sh '''#!/bin/bash -x
                            tagRegex='^v[0-9]+\\.[0-9]+\\.[0-9]+-rc[0-9]+$'
                            if [[ ${tag} =~ $tagRegex ]]; then
                                echo "Valid pre-production release tag: ${tag}"
                            else
                                echo "Invalid tag for production release promotion: ${tag} -- should match format 'v2.1.52-rc2' etc"
                                exit 1
                            fi
                            git checkout ${tag}
                            git checkout -b ${binTag}-release
                            git config --global user.email "mgmt@nativeplanet.io"
                            git config --global user.name "Native Planet CICD"
                            git config --global credential.helper store && echo "https://${npGhToken}:x-oauth-basic@github.com" > ~/.git-credentials
                            sed -i "4s/.*/export const version = writable(\\"${binTag}\\")/" ./ui/src/lib/stores/display.js
                            sed -i "11s/.*/TAG=${binTag}/" ./release/groundseg_install.sh
                            version_defaults="./goseg/defaults/version.go"
                            json_blob=$(curl -s https://version.groundseg.app)
                            formatted_json_blob=$(echo "$json_blob" | jq '.')
                            start_line=$(grep -n 'DefaultVersionText =' "$version_defaults" | cut -d ':' -f1)
                            end_line=$(grep -n 'VersionInfo' "$version_defaults" | cut -d ':' -f1)
                            temp_file=$(mktemp)
                            head -n $((start_line-1)) "$version_defaults" > "$temp_file"
                            echo "  DefaultVersionText = \\`" >> "$temp_file"
                            echo "$formatted_json_blob" >> "$temp_file"
                            echo "\\`" >> "$temp_file"
                            tail -n +$end_line -q "$version_defaults" >> "$temp_file"
                            mv "$temp_file" "$version_defaults"
                            cd ./ui
                            DOCKER_BUILDKIT=0 docker build -t web-builder -f builder.Dockerfile .
                            container_id=$(docker create web-builder)
                            docker cp $container_id:/webui/build ./web
                            rm -rf ../goseg/web
                            mv web ../goseg/

                            DOCKER_BUILDKIT=0 docker build -t web-builder -f gallseg.Dockerfile .
                            container_id=$(docker create web-builder)
                            git clone https://github.com/Native-Planet/globber
                            cd globber
                            docker cp $container_id:/webui/build ./web
                            ./glob.sh web

                            hash=$(ls -1 -c . | head -1 | sed "s/glob-\\([a-z0-9\\.]*\\).glob/\\1/")
                            globurl="https://files.native.computer/glob/gallseg-${tag}-${hash}.glob"
                            echo "hash=${hash}" > /opt/groundseg/version/glob/globhash.env
                            echo ${globurl} > /opt/groundseg/version/glob/globurl.txt

                            mv ./*.glob /opt/groundseg/version/glob/gallseg-${tag}-${hash}.glob
                            cd ..
                            rm -rf globber
                            cd ..
                            docketinfo="    glob-http+['${globurl}' ${hash}]"
                            sed -i "/glob-http/c\${docketinfo}" gallseg/desk.docket-0
                            echo "~nattyv" > gallseg/desk.ship
                            
                            rm -r /opt/groundseg/release-ships/latest/groundseg
                            cp -r gallseg /opt/groundseg/release-ships/latest/groundseg
                            /opt/groundseg/release-ships/click/click -k -i /opt/groundseg/release-ships/commit.hoon /opt/groundseg/release-ships/latest

                            cd goseg
                            go fmt ./...
			    go mod tidy
                            git commit -am "Promoting ${binTag} for release"
                            git tag ${binTag}
                            git push --set-upstream origin ${binTag}
                            git push --tags
                        '''
                        sh '''#!/bin/bash -x
			    cd goseg
                            env GOOS=linux CGO_ENABLED=0 GOARCH=amd64 go build -o /opt/groundseg/version/bin/groundseg_amd64_${binTag}_latest
                            env GOOS=linux CGO_ENABLED=0 GOARCH=arm64 go build -o /opt/groundseg/version/bin/groundseg_arm64_${binTag}_latest
                        '''
                    }
                }
            }
        }
        stage('move binaries') {
            steps {
                script {
                    /* copy to r2 */
                    if( "${params.CHANNEL}" != "nobuild" ) {  
                        sh 'echo "debug: post-build actions"'
                        sh """#!/bin/bash -x
                        rclone -vvv --config /var/jenkins_home/rclone.conf copy /opt/groundseg/version/bin/groundseg_arm64_${env.binTag}_${env.channel} r2:groundseg/bin
                        rclone -vvv --config /var/jenkins_home/rclone.conf copy /opt/groundseg/version/bin/groundseg_amd64_${env.binTag}_${env.channel} r2:groundseg/bin
                        """
                        sh '''#!/bin/bash -x
                        source /opt/groundseg/version/glob/globhash.env
                        rclone -vvv --config /var/jenkins_home/rclone.conf copy /opt/groundseg/version/glob/gallseg-${tag}-${hash}.glob r2:groundseg/glob
                        '''
                        /*
                        */
                    }
                }
            }
        }
        stage('version update') {
            environment {
                channel = "${params.CHANNEL}"
                /* update versions and hashes on public version server */
                def armsha = sh(
                    script: """#!/bin/bash -x
                        val=`sha256sum /opt/groundseg/version/bin/groundseg_arm64_${env.binTag}_${env.channel} | awk '{print \$1}'`
                        echo \${val}
                    """,
                    returnStdout: true
                ).trim()
                def amdsha = sh(
                    script: """#!/bin/bash -x
                        val=`sha256sum /opt/groundseg/version/bin/groundseg_amd64_${env.binTag}_${env.channel} |awk '{print \$1}'`
                        echo \${val}
                    """,
                    returnStdout: true
                ).trim()
                major = sh(
                    script: '''#!/bin/bash -x
                        ver=${tag}
                        if [[ "${tag}" == *"-"* ]]; then
                            ver=`echo ${tag}|awk -F '-' '{print \$1}'`
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
                            ver=`echo ${tag}|awk -F '-' '{print \$1}'`
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
                            ver=`echo ${tag}|awk -F '-' '{print \$1}'`
                        fi
                        patch=`echo ${ver}|awk -F '.' '{print \$3}'|sed 's/v//g'`
                        echo ${patch}
                    ''',
                    returnStdout: true
                ).trim()
                armbin = "https://files.native.computer/bin/groundseg_arm64_${env.binTag}_${channel}"
                amdbin = "https://files.native.computer/bin/groundseg_amd64_${env.binTag}_${channel}"
            }
            steps {
                script {
                    def to_canary = "${params.TO_CANARY}".toLowerCase()
                    if( "${params.CHANNEL}" == "latest" ) {
                        sh '''#!/bin/bash -x
                            cp ./release/standard_install.sh /opt/groundseg/get/install.sh
                            cp ./release/groundseg_install.sh /opt/groundseg/get/only.sh
                            webui_amd64_hash=`curl https://${VERSION_SERVER} | jq -r '.[].edge.webui.amd64_sha256'`
                            webui_arm64_hash=`curl https://${VERSION_SERVER} | jq -r '.[].edge.webui.arm64_sha256'`
                            curl -X PUT -H "X-Api-Key: ${versionauth}" -H 'Content-Type: application/json' \
                                https://${VERSION_SERVER}/modify/groundseg/latest/groundseg/amd64_url/payload \
                                -d "{\\"value\\":\\"${amdbin}\\"}"
                            curl -X PUT -H "X-Api-Key: ${versionauth}" -H 'Content-Type: application/json' \
                                https://${VERSION_SERVER}/modify/groundseg/latest/groundseg/arm64_url/payload \
                                -d "{\\"value\\":\\"${armbin}\\"}"
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://${VERSION_SERVER}/modify/groundseg/latest/groundseg/amd64_sha256/${amdsha}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://${VERSION_SERVER}/modify/groundseg/latest/groundseg/arm64_sha256/${armsha}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://${VERSION_SERVER}/modify/groundseg/latest/groundseg/major/${major}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://${VERSION_SERVER}/modify/groundseg/latest/groundseg/minor/${minor}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://${VERSION_SERVER}/modify/groundseg/latest/groundseg/patch/${patch}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" -H 'Content-Type: application/json' \
                                https://${VERSION_SERVER}/modify/groundseg/canary/groundseg/amd64_url/payload \
                                -d "{\\"value\\":\\"${amdbin}\\"}"
                            curl -X PUT -H "X-Api-Key: ${versionauth}" -H 'Content-Type: application/json' \
                                https://${VERSION_SERVER}/modify/groundseg/canary/groundseg/arm64_url/payload \
                                -d "{\\"value\\":\\"${armbin}\\"}"
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://${VERSION_SERVER}/modify/groundseg/canary/groundseg/amd64_sha256/${amdsha}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://${VERSION_SERVER}/modify/groundseg/canary/groundseg/arm64_sha256/${armsha}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://${VERSION_SERVER}/modify/groundseg/canary/groundseg/major/${major}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://${VERSION_SERVER}/modify/groundseg/canary/groundseg/minor/${minor}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://${VERSION_SERVER}/modify/groundseg/canary/groundseg/patch/${patch}
                        '''
                    }
                    if( "${params.CHANNEL}" == "edge" ) {
                        sh '''#!/bin/bash -x
                            curl -X PUT -H "X-Api-Key: ${versionauth}" -H 'Content-Type: application/json' \
                                https://${VERSION_SERVER}/modify/groundseg/edge/groundseg/amd64_url/payload \
                                -d "{\\"value\\":\\"${amdbin}\\"}"
                            curl -X PUT -H "X-Api-Key: ${versionauth}" -H 'Content-Type: application/json' \
                                https://${VERSION_SERVER}/modify/groundseg/edge/groundseg/arm64_url/payload \
                                -d "{\\"value\\":\\"${armbin}\\"}"
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://${VERSION_SERVER}/modify/groundseg/edge/groundseg/amd64_sha256/${amdsha}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://${VERSION_SERVER}/modify/groundseg/edge/groundseg/arm64_sha256/${armsha}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://${VERSION_SERVER}/modify/groundseg/edge/groundseg/major/${major}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://${VERSION_SERVER}/modify/groundseg/edge/groundseg/minor/${minor}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://${VERSION_SERVER}/modify/groundseg/edge/groundseg/patch/${patch}
                        '''
                    }
                    if( "${params.TO_CANARY}" == "true" ) {
                        sh '''#!/bin/bash -x
                            curl -X PUT -H "X-Api-Key: ${versionauth}" -H 'Content-Type: application/json' \
                                https://${VERSION_SERVER}/modify/groundseg/canary/groundseg/amd64_url/payload \
                                -d "{\\"value\\":\\"${amdbin}\\"}"
                            curl -X PUT -H "X-Api-Key: ${versionauth}" -H 'Content-Type: application/json' \
                                https://${VERSION_SERVER}/modify/groundseg/canary/groundseg/arm64_url/payload \
                                -d "{\\"value\\":\\"${armbin}\\"}"
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://${VERSION_SERVER}/modify/groundseg/canary/groundseg/amd64_sha256/${amdsha}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://${VERSION_SERVER}/modify/groundseg/canary/groundseg/arm64_sha256/${armsha}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://${VERSION_SERVER}/modify/groundseg/canary/groundseg/major/${major}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://${VERSION_SERVER}/modify/groundseg/canary/groundseg/minor/${minor}
                            curl -X PUT -H "X-Api-Key: ${versionauth}" \
                                https://${VERSION_SERVER}/modify/groundseg/canary/groundseg/patch/${patch}
                        '''
                    }
                }
            }
        }
        stage('SonarQube') {
            environment {
                scannerHome = "${tool 'SonarQubeScanner'}"
            }
            steps {
                script {
                    if( "${params.CHANNEL}" == "edge" ) {
                        withSonarQubeEnv('SonarQube') {
                            sh "${scannerHome}/bin/sonar-scanner -Dsonar.projectKey=Native-Planet_GroundSeg_AYZoKNgHuu12TOn3FQ6N -Dsonar.sources=./goseg"
                        }
                    }
                }
            }
        }
        stage('merge to master') {
            steps {
                /* merge tag changes into master if deploying to master */
                script {
                    if(( "${params.CHANNEL}" == "latest" ) && ( "${params.PROMOTE}" == "promote" )) {
                        withCredentials([gitUsernamePassword(credentialsId: 'Github token', gitToolName: 'Default')]) {
                            sh (
                                script: '''#!/bin/bash -x
                                    git remote update
                                    git fetch
                                    git checkout --track origin/master
                                    git merge ${binTag} -m "Merged ${tag}"
                                    git config --global --add --bool push.autoSetupRemote true
                                    git push
                                '''
                            )
                        }
                    }
                }
            }
        }
        stage('github release') {
            steps {
                script {
                    if(( "${params.CHANNEL}" == "latest" ) && ( "${params.VERSION_SERVER}" != "staging.version.groundseg.app")) {
			            sh (
                            script: '''#!/bin/bash -x
                                MESSAGE="Release ${binTag}"
                                VERSION=$(echo "${binTag}"|sed "s/v//g")
                                API_JSON="{\\"tag_name\\": \\"${binTag}\\",\\"target_commitish\\": \\"master\\",\\"name\\": \\"${binTag}\\",\\"body\\": \\"${MESSAGE}\\",\\"draft\\": false,\\"prerelease\\": false}"
                                API_RESPONSE_STATUS=$(curl -H "Authorization: token ${npGhToken}" --data "$API_JSON" -s -i "https://api.github.com/repos/Native-Planet/GroundSeg/releases")
                                echo "Release: ${API_RESPONSE_STATUS}"
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
        success {
            script {
                glob_url = readFile('/opt/groundseg/version/glob/globurl.txt').trim()
                currentBuild.description = "Glob URL: ${glob_url}"
            }
        }
    }
}
