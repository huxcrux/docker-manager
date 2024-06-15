# docker-manager

This is a ligthweigth golang program that:

* Allows you to specify what containers you wish to run
* Removes unwanted containers (if configured)
* Ensures container config are correct
* Can check for new image releases

This program is brand new and bug most likely exists.

## Usage

1. Create a config (example below)
2. Run the program

## Example config

```yaml
app_config:
  debug: True
  update_check: True
  remove_unwanted_containers: True

containers:
  - name: nginx_1
    image: nginx:latest
    port_bindings:
      - port: 80
        protocol: tcp
        host_ip: 0.0.0.0
        host_port: 8090
    env:
      - key1=value1
```

## Scale

This is currently a bit unclear. I have tested with 1 and 10 containers and the service is using around 12MB of RAM.

## known issues

* Environent variables not compared (no update if changed)
* Very few options can be set on containers. This is currently by design to get a MWP ready
