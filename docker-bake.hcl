variable "REGISTRY" {
  default = "us-docker.pkg.dev"
}

variable "REPOSITORY" {
  default = "oplabs-tools-artifacts/images"
}

variable "GIT_COMMIT" {
  default = "dev"
}

variable "GIT_DATE" {
  default = "0"
}

variable "IMAGE_TAGS" {
  default = "${GIT_COMMIT}" // split by ","
}

variable "PLATFORMS" {
  // You can override this as "linux/amd64,linux/arm64".
  // Only a specify a single platform when `--load` ing into docker.
  // Multi-platform is supported when outputting to disk or pushing to a registry.
  // Multi-platform builds can be tested locally with:  --set="*.output=type=image,push=false"
  default = "linux/amd64"
}

target "op-monitorism" {
  dockerfile = "Dockerfile"
  context = "./op-monitorism"
  args = {
    GITCOMMIT = "${GIT_COMMIT}"
    GITDATE = "${GIT_DATE}"
  }
  platforms = split(",", PLATFORMS)
   tags = [
    for tag in split(",", IMAGE_TAGS) : "${REGISTRY}/${REPOSITORY}/op-monitorism:${tag}",
    if GIT_VERSION != "" && GIT_VERSION != "untagged" "${REGISTRY}/${REPOSITORY}/op-monitorism:${GIT_VERSION}"
  ]
}
