# drone-openapi

[![Build Status](https://cloud.drone.io/api/badges/nytimes/drone-openapi/status.svg)](https://cloud.drone.io/nytimes/drone-openapi)

Publish Open API spec files from a Drone pipeline.

## Links

- Docker Hub [release tags](https://hub.docker.com/r/nytimes/drone-openapi/tags)
- Drone.io [builds](https://cloud.drone.io/nytimes/drone-openapi)
- Contributing [documentation](.github/CONTRIBUTING.md)

## Overview

This plugin accepts a team name along with a valid Open API spec file or directory of files, and it will post the given information to the given `uploader_url`.

## Drone versions

This plugin supports Drone 1.0+.

The example below is for secrets in the Drone 1.0+ format, where the GCP Service Account json must be passed to the `GOOGLE_CREDENTIALS` parameter in `.drone.yml` as an environment variable.

## Usage

Basic example config to publish the swagger.yaml spec file under the kids team

```yaml
  - name: publish-openapi
    image: nytimes/drone-openapi
    settings:
      uploader_url: https://apis.nyt.net/update
      spec: swaggerui/swagger.yaml
      team: kids
    environment:
      GOOGLE_CREDENTIALS:
        from_secret: GOOGLE_CREDENTIALS
    when:
      event:
      - push
      branch:
      - main
```

To publish all yamls in a given directory, specify a `specs_dir` with your yamls within it:

```yaml
  - name: publish-openapi
    image: nytimes/drone-openapi
    settings:
      uploader_url: https://apis.nyt.net/update
      specs_dir: swaggerui/
      team: kids
    environment:
      GOOGLE_CREDENTIALS:
        from_secret: GOOGLE_CREDENTIALS
    when:
      event:
      - push
      branch:
      - main
```
