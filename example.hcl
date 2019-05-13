service "svc1-staging" {
  repository = "mynamespace/svc-1"
  conditions = "refs/heads/master"
  cmd        = "docker run --rm --name svc1-staging alpine echo 'deployed svc1-staging:latest'"
}

service "svc1-production" {
  repository = "mynamespace/svc-1"
  conditions = "refs/tags/(.+)"
  cmd        = "docker run --rm --name svc1-production alpine echo 'deployed svc1-production:$1'"
}
