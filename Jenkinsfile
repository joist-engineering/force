node('docker') {
  def branchName = env.BRANCH_NAME.replaceAll(/[^a-zA-Z0-9\s\:]/, '_')
  def buildNumber = env.BUILD_NUMBER

  try {
    stage 'Build base container'
    checkout scm
    def forceDockerImageName = "joist-engineering/force:${branchName}_${buildNumber}"
    sh "docker build -t ${forceDockerImageName} ."

    stage 'Run tests'
    sh "docker run -v \$(pwd):/go/src/github.com/joist-engineering/force ${forceDockerImageName} go test"

    stage 'Build binaries for Linux and OS X on x86_64'
    sh "docker run -v \$(pwd):/go/src/github.com/joist-engineering/force ${forceDockerImageName} gox -os \"darwin linux\" -arch=amd64"

    step([$class: 'ArtifactArchiver', artifacts: 'force_darwin_amd64,force_linux_amd64', fingerprint: true])

    if (env.BRANCH_NAME == "integration") {
      stage 'Upload build of integration branch to "dev" version of update site'
      // TODO
    }

    if (env.BRANCH_NAME == "master") {
      stage 'Upload build of master branch to "stable" version of update site'
      // TODO
    }
  } finally {
    stage 'Clean up Docker containers and images'
    // clean up *all* (even ones not from this Jenkins job) dangling containers and images, best effort.
    // aggressive error handling to deal with mis-designs in the docker CLI mean some reasonable cases of the above will produce errors as a side-effect.
    try { sh "docker stop `docker ps -a -q -f status=exited`" } catch (e) {}
    try { sh "docker rm -v `docker ps -a -q -f status=exited`" } catch (e) {}
    // and remove our tagged image:
    try { sh "docker rmi ${forceDockerImageName}" } catch (e) {}
    try { sh "docker rmi `docker images --filter 'dangling=true' -q --no-trunc`" } catch (e) {}
  }
}
