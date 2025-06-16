def ecr_token
def BUILD_ENVIRONMENT
def ECR_CREDENTIALS

pipeline {
    agent { label 'arm-node' }
    environment {
        BUILDER_NAME = 'k8s-builder'
        DOCKER_REGISTRY = '578061096415.dkr.ecr.us-east-1.amazonaws.com'              // Replace with your AWS account ID                 // Replace with your region
        APP_NAME = 'calert'                   // Replace with your app name
        DOCKER_IMAGE = "${DOCKER_REGISTRY}/${APP_NAME}:${VERSION}"
        GIT_REPO_URL = 'https://github.com/mykaarma/calert.git'
    }
    parameters {
        string(name: 'branch', defaultValue: 'main')
    }

    stages {
        stage("Source Code Management"){
            steps {
                script {
                    checkout([
                        $class: 'GitSCM',
                        branches: [[name: "${params.branch}"]],
                        userRemoteConfigs: [[
                            url: "${GIT_REPO_URL}",
                            credentialsId: 'git_credentials'
                        ]]
                    ])
                }
            }
        }
        // ADDED: New stage to prepare dynamic environment variables
        stage('Prepare Environment') {
            steps {
                script {
                    // Read the version from the file in your repo
                    def version = readFile('version.txt').trim()
                    echo "Building version: ${version}"

                    // Set environment variables that will be available in all subsequent stages
                    env.VERSION = version
                    env.DOCKER_IMAGE_TAG = "${DOCKER_REGISTRY}/${APP_NAME}:${version}"
                }
            }
        }

        stage('Build Go App') {
            steps {
               sh 'docker run --rm -v "$PWD":/app -w /app golang:1.21 sh -c "GOOS=linux GOARCH=arm64 go build -v -buildvcs=false -o cmd/calert.bin ./cmd"'
            }
        }

        // Stage 4: Build the Docker image
        stage('Build Docker Image') {
            steps {
                echo "Verifying workspace before building Docker image..."
                sh 'pwd'
                sh 'ls -la'

                // UPDATED: Use the dynamically created image tag
                echo "Building Docker image: ${env.DOCKER_IMAGE_TAG}"
                script {
                    def dockerImage = docker.build("${env.DOCKER_IMAGE_TAG}", ".")
                }
            }
        }

         stage('Login and Push to ECR') {
            steps {
                script {
                    // --- Your Dynamic Credential Logic ---
                    def jenkinsUrl = env.JENKINS_URL ?: '' // Use the reliable, built-in variable
                    echo "Using  Jenkins as: ${jenkinsUrl}"

                    if (jenkinsUrl.contains("qa-ci")) {
                        echo "Detected QA environment."
                        ECR_CREDENTIALS = 'qa-ecr'
                    } else {
                        echo "Detected Prod environment."
                        ECR_CREDENTIALS = 'ecr'
                    }
                    echo "Using ECR credentials stored in Jenkins as: '${ECR_CREDENTIALS}'"

                    // --- The Recommended Way to Log In and Push ---
                    // This block securely handles the ECR login and logout.
                    // The 'ecrCredentialsId' must match the ID of your credentials in Jenkins.
                    withCredentials([aws(credentialsId: "${ECR_CREDENTIALS}")]){
                            sh "env"
                            ecr_token = sh(script: "aws ecr get-login-password --region us-east-1", returnStdout: true).trim()
                    }
                }
            }
        }

        stage('Push to ECR') {
            steps {
                 sh """
                     docker login --username AWS --password ${ecr_token} ${DOCKER_REGISTRY}
                """
                sh "docker push ${env.DOCKER_IMAGE_TAG}"
            }
        }
    }
}