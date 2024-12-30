pipeline {
    agent none
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
                    go build
                    '''
                }
                archiveArtifacts artifacts: 'cacheserver', fingerprint: true, followSymlinks: true, onlyIfSuccessful: true
            }
        }

        stage('Build on Windows') {
            agent {
                label 'windows'
            }
            steps {
                script {
                    bat '''
                    echo "Building Go project on Windows"
                    go version
                    go build
                    '''
                }
            }
            archiveArtifacts artifacts: 'cacheserver.exe', fingerprint: true, followSymlinks: true, onlyIfSuccessful: true
        }
    }
}
