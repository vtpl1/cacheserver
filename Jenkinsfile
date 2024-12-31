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
                    make
                    '''
                }
                archiveArtifacts artifacts: 'bin/*', fingerprint: true, followSymlinks: true, onlyIfSuccessful: true
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
                    '''
                }
                sh 'pwsh build.ps1'
                archiveArtifacts artifacts: 'binwin\\*', fingerprint: true, followSymlinks: true, onlyIfSuccessful: true
            }            
        }
    }
}
