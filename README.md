# drone-openapi

### Publish Open API spec files from a Drone pipeline

This plugin accepts a team name along with a valid Open API spec file and it will post the given information to the given uploader_url.

### Drone versions

This plugin supports Drone 0.4 and 0.6+ (0.5 is deprecated).

The example below is for secrets in the Drone 1.0+ format, where the GCP Service Account json must be passed to the `GOOGLE_CREDENTIALS` parameter in .drone.yml as an environment variable.


### Basic example config to publish the swagger.yaml spec file under the kids team:

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
