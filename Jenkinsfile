node('docker') {
  stage 'Build base container'
  checkout scm
  sh "docker build -t joist-engineering/force:latest ."

  stage 'Run tests'
  sh "docker run -v \$(pwd):/go/src/github.com/joist-engineering/force joist-engineering/force:latest glide install"
  sh "docker run -v \$(pwd):/go/src/github.com/joist-engineering/force joist-engineering/force:latest go test"
  sh "echo poo poo butt"

  stage 'Build binaries for Linux and OS X on x86_64'
  sh "docker run -v \$(pwd):/go/src/github.com/joist-engineering/force joist-engineering/force:latest gox -os \"darwin linux\" -arch=amd64"

  step([$class: 'ArtifactArchiver', artifacts: 'force_darwin_amd64,force_linux_amd64', fingerprint: true])

  if (env.BRANCH_NAME == "integration") {
    stage 'Upload build of integration branch to "dev" version of update site'

  }

  if (env.BRANCH_NAME == "master") {
    stage 'Upload build of master branch to "stable" version of update site'
  }
}
