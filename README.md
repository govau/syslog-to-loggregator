# syslog_to_loggregator

Start a syslog server, and forward to [loggregator](https://github.com/cloudfoundry/loggregator).

Uses the [Go Client Library for Loggregator](https://github.com/cloudfoundry/go-loggregator).

Notes
https://github.com/cloudfoundry/loggregator/issues/19
https://github.com/frodenas/syslog-to-metron
https://github.com/cloudfoundry/bosh-hm-forwarder-release
https://github.com/cloudfoundry-incubator/haproxy-boshrelease/issues/81
https://github.com/jeromer/syslogparser
syslog to file? - http://kvz.io/blog/2010/08/11/haproxy-logging/

## Usage

syslog_to_loggregator is written to be colocated with a metron_agent, and uses
the dropsonde protocol.

## Development

You can run this locally with bosh lite and cf-deployment.

In these instructions, we add haproxy onto the router instances group which logs to our syslog server.

We assume all git clones occur in `~/workspace`.

### BOSH lite

```
git clone https://github.com/cloudfoundry/bosh-deployment
cd bosh-deployment

# This is idempotent, so run again if needs to change
bosh create-env ./bosh.yml \
    --state ./state.json \
    -o ./virtualbox/cpi.yml \
    -o ./virtualbox/outbound-network.yml \
    -o ./bosh-lite.yml \
    -o ./bosh-lite-runc.yml \
    -o ./jumpbox-user.yml \
    -o ./uaa.yml \
    -o ./credhub.yml \
    --vars-store ./creds.yml \
    -v director_name="main" \
    -v internal_ip=192.168.50.6 \
    -v internal_gw=192.168.50.1 \
    -v internal_cidr=192.168.50.0/24 \
    -v outbound_network_name=NatNetwork
bosh alias-env vbox -e 192.168.50.6 --ca-cert <(bosh int ./creds.yml --path /director_ssl/ca)

```

### cf-deployment

To keep it simple initially, first deploy cf-deployment:

```
git clone https://github.com/cloudfoundry/cf-deployment
cd cf-deployment

# Upload stemcell (only run if needed)
bosh upload-stemcell "https://s3.amazonaws.com/bosh-core-stemcells/warden/bosh-stemcell-$(bosh int cf-deployment.yml --path /stemcells/alias=default/version)-warden-boshlite-ubuntu-trusty-go_agent.tgz"

# Set cloud config
bosh update-cloud-config iaas-support/bosh-lite/cloud-config.yml

# Deploy CloudFoundry (idempotent)
bosh -d cf deploy -n cf-deployment.yml \
    -o operations/bosh-lite.yml \
    -o operations/use-compiled-releases.yml \
    -v system_domain=bosh-lite.com
```

Add network routes:

```
# Mac OS X
sudo route add -net 10.244.0.0/16 192.168.50.6

# If it still doesn't work:
sudo route delete -net 10.244.0.0/16
sudo route add -net 10.244.0.0/16 192.168.50.6
```

Verify it is up (resolves to a local address):

```
curl api.bosh-lite.com
```

Login to the api:

```
# Login to credhub
credhub login -s https://192.168.50.6:8844 -u "credhub-cli" -p "$(bosh int "~/workspace/bosh-deployment/creds.yml" --path /credhub_cli_password)" --skip-tls-validation

# Get the cf admin password
credhub g -n /main/cf/cf_admin_password

# login to cf
cf login -a https://api.bosh-lite.com --skip-ssl-validation

# The username is admin, and use the above password from credhub
```

### cf-deployment with dev release of syslog_to_loggregator

Now you've made it this far, we can re-deploy cf-deployment with a local dev release of syslog_to_loggregator.

We add the DTA [frontend-boshrelease](https://github.com/govau/frontend-boshrelease) which adds an haproxy to
the router instance_groups which logs to our syslog server.

```

cd ~/workspace/syslog-to-loggregator-boshrelease
bosh create-release --name=syslog_to_loggregator --force
bosh upload-release

bosh -d cf deploy -n ~/workspace/cf-deployment/cf-deployment.yml \
    -o ~/workspace/cf-deployment/operations/bosh-lite.yml \
    -o ~/workspace/cf-deployment/operations/use-compiled-releases.yml \
    -v system_domain=bosh-lite.com \
    -o manifests/decrease-canary-watch-time.yml \
    -o manifests/add-frontend.yml \
    -v leresponder_external_hostname=notused.local \
    -o manifests/add-syslog-to-loggregator.yml
```

Haproxy is now running on the router instance on port 1080, and its logs are being sent to a syslog server which are then forwarded to loggregator.

We can verify this using the
[CF Nozzle plugin](https://github.com/cloudfoundry-attic/firehose-plugin) by starting a nozzle and curl'ing haproxy.

Watch all log messages using the nozzle plugin:

```
cf add-plugin-repo CF-Community http://plugins.cloudfoundry.org/
cf install-plugin "Firehose Plugin" -r CF-Community
# assumes you are already logged in with the admin user
cf nozzle -filter LogMessage
```

In another terminal, curl haproxy:

```
# Find the IP of the router instance
bosh vms |grep router

# GET something from haproxy
curl -I http://IP:1080
```

You should now see logs for each GET request in the nozzle plugin terminal.
