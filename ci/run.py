#!/usr/bin/env python3
#
# Copyright 2019-2020 Jean-Francois Smigielski
#
# This software is supplied under the terms of the MIT License, a
# copy of which should be located in the distribution where this
# file was obtained (LICENSE.txt). A copy of the license may also be
# found online at https://opensource.org/licenses/MIT.
#

import os
import subprocess
from string import Template
import time
import tempfile

BASEDIR = tempfile.mkdtemp(suffix='-test', prefix='gunkan-')
DATADIR = BASEDIR + "/data"
CFGDIR = BASEDIR + "/etc"

DC = "dc0"
ip="127.0.0.1"

consul_tpl = Template("""{
    "node_name": "test-node",
    "datacenter": "test-dc",
    "data_dir": "$vol",
    "log_level": "INFO",
    "server": true,
    "enable_syslog": true,
    "syslog_facility": "LOCAL0",
    "ui": true,
    "serf_lan": "$ip",
    "serf_wan": "$ip",
    "bind_addr": "$ip",
    "client_addr": "$ip"
}""")


consul_srv_http_tpl = Template("""{
    "service": {
        "check": {
            "id": "$id",
            "interval": "2s",
            "timeout": "1s",
            "http": "http://$ip:$port/health"
        },
        "id": "$id",
        "name": "$type",
        "tags": [ "$tag" ],
        "address": "$ip",
        "port": $port
    }
}""")

consul_srv_grpc_tpl = Template("""{
    "service": {
        "checks": [
            { "interval": "2s", "timeout": "1s", "tcp": "$ip:$port" }
        ],
        "id": "$id",
        "name": "$type",
        "tags": [ "$tag" ],
        "address": "$ip",
        "port": $port
    }
}""")


def stateless(t, num, e):
    uid = t + '-' + str(num)
    return {"tag": t, "type": t, "id": uid,
            "ip": ip, "port": 6000 + num, "exe": e,
            "vol": None, "cfg": CFGDIR}


def statefull(t, num, e):
    uid = t + '-' + str(num)
    return {"tag": t, "type": uid, "id": uid,
            "ip": ip, "port": 6000 + num, "exe": e,
            "vol": DATADIR + '/' + t + '-' + str(num), "cfg": CFGDIR}


def do(*args):
    subprocess.check_call(args)


def generate_certificate(path):
    def rel(s):
        return path + '/' + s
    with open(rel('certificate.conf'), 'w') as f:
        f.write('''
[ req ]
prompt = no
default_bits = 4096
distinguished_name = req_distinguished_name
req_extensions = req_ext

[ req_distinguished_name ]
C=FR
ST=Nord
L=Hem
O=OpenIO
OU=R&D
CN=localhost

[ req_ext ]
subjectAltName = @alt_names

[alt_names]
DNS.1 = hostname.domain.tld
DNS.2 = hostname
IP.1 = 127.0.0.1
''')
    do('openssl', 'genrsa',
       '-out', rel('ca.key'), '4096')
    do('openssl', 'req', '-new',
       '-x509', '-key', rel('ca.key'), '-sha256',
       '-subj', "/C=FR/ST=Nord/O=CA, Inc./CN=localhost", '-days', '365',
       '-out', rel('ca.cert'))
    do('openssl', 'genrsa',
       '-out', rel('service.key'), '4096')
    do('openssl', 'req', '-new',
       '-key', rel('service.key'), '-out', rel('service.csr'),
       '-config', rel('certificate.conf'))
    do('openssl', 'x509', '-req',
       '-in', rel('service.csr'),
       '-CA', rel('ca.cert'), '-CAkey', rel('ca.key'), '-CAcreateserial',
       '-out', rel('service.pem'), '-days', '365', '-sha256',
       '-extfile', rel('certificate.conf'),
       '-extensions', 'req_ext')
    do('openssl', 'x509',
       '-in', rel('service.pem'), '-text', '-noout')


def sequence(start=0):
    while True:
        yield start
        start+=1


def services():
    port = sequence()
    for _i in range(3):
        yield "grpc", statefull("gkindex-store", next(port), "gunkan-index-store-rocksdb")
    for _i in range(3):
        yield "http", statefull("gkblob-store", next(port), "gunkan-blob-store-fs")
    for _i in range(3):
        yield "grpc", stateless("gkindex-gate", next(port), "gunkan-index-gate")
    for _i in range(3):
        yield "http", stateless("gkdata-gate", next(port), "gunkan-data-gate")


# Create the working directories
for kind, srv in services():
    try:
        if srv['vol']:
            os.makedirs(srv['vol'])
    except OSError:
        pass
    try:
        if srv['cfg']:
            os.makedirs(srv['cfg'])
    except OSError:
        pass

# Populate the consul configuration
try:
    os.makedirs(CFGDIR + '/consul-0.d')
except OSError:
    pass

with open(CFGDIR + '/consul-0.json', 'w') as f:
    f.write(consul_tpl.safe_substitute(**{'vol': DATADIR + '/consul-0', 'ip': ip}))

for kind, srv in services():
    with open(CFGDIR + '/consul-0.d/srv-' + srv['id'] + '.json', 'w') as f:
        if kind =='http':
            f.write(consul_srv_http_tpl.safe_substitute(**srv))
        elif kind == 'grpc':
            f.write(consul_srv_grpc_tpl.safe_substitute(**srv))
        else:
            raise ValueError("Invalid service kind")

# Generate a certificate that will be used by all the services.
generate_certificate(CFGDIR)

# Start the services
children = list()
for kind, srv in services():
    endpoint = srv['ip'] + ':' + str(srv['port'])
    cmd = [srv['exe'], '--tls', srv['cfg'], endpoint]
    if srv['vol']:
        cmd.append(srv['vol'])
    print(repr(cmd))
    child = subprocess.Popen(cmd)
    children.append(child)

consul = subprocess.Popen((
        'consul', 'agent', '-server', '-bootstrap', '-dev', '-ui',
        '-config-file', CFGDIR + '/consul-0.json',
        '-config-dir', CFGDIR + '/consul-0.d'))
children.append(consul)

# Wait for a termination event
try:
    while True:
        time.sleep(1.0)
except:
    pass

# Final cleanup
for child in children:
    child.terminate()
for child in children:
    child.wait()
