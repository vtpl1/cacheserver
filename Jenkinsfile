pipeline {
    agent none
    environment {
        DOCKERHUB_CREDENTIALS = credentials('Jenkins_build')
    }
    stages {
        stage('Build on Linux with Docker') {
            agent {
                docker {
                    label 'linux'
                    image 'golang:latest'
                }
            }
            steps {
                script {
                    sh '''
                    echo "Building Go project on Linux"
                    go version
                    make
                    '''
                }
                archiveArtifacts artifacts: 'bin/*', fingerprint: true, followSymlinks: true, onlyIfSuccessful: true
            }
        }

        // stage('Build on Windows') {
        //     agent {
        //         label 'windows'
        //     }
        //     steps {
        //         script {
        //             bat '''
        //             echo "Building Go project on Windows"
        //             go version                    
        //             '''
        //         }
        //         powershell '.\\build.ps1'
        //         archiveArtifacts artifacts: 'binwin\\*', fingerprint: true, followSymlinks: true, onlyIfSuccessful: true
        //     }            
        // }

        stage('Create go docker image') {
            agent {
                node {
                    label "linux"
                }
            }
            steps {
                sh 'rm -rf bin || true'
                dir('bin') {
                    copyArtifacts projectName: "${JOB_NAME}", filter: "bin/cacheserver_linux_amd64", fingerprintArtifacts: true, flatten: false, selector: specific("${BUILD_NUMBER}");
                }
                script {
                    sh 'rm -f temp.Dockerfile || true'
                    writeFile(file: 'temp.Dockerfile', text: readFile(file: 'prod.Dockerfile'))
                    PACKAGE_NAME = "cache_service"
                    IMAGE_NAME = "cache_service"
                    TAG = "unstable"
                    println "vtpl/${IMAGE_NAME}:${TAG}"
                    println "${TAG}"
                    sh 'echo $DOCKERHUB_CREDENTIALS_PSW | docker login -u $DOCKERHUB_CREDENTIALS_USR --password-stdin'
                    prodBuild = docker.build("vtpl/${IMAGE_NAME}:${TAG}",
                        "--build-arg PACKAGE_NAME:${PACKAGE_NAME} -f temp.Dockerfile .")
                    prodBuild.push()
                    sh 'rm -f temp.Dockerfile || true'
                }
            }
        }
    }
}
