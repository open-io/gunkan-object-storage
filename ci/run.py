#!/usr/bin/env python
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

BASEDIR = "/tmp/gunkan"
DATADIR = BASEDIR + "/data"
CFGDIR = BASEDIR + "/etc"

DC = "dc0"
ip="127.0.0.1"

consul_tpl = Template("""{
    "node_name": "test-node",
    "datacenter": "tes-dc",
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
            "id": "check-$id",
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
        "check": {
            "id": "check-$id",
            "interval": "2s",
            "timeout": "1s",
            "grpc_use_tls": true,
            "grpc": "$ip:$port"
        },
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
            "vol": None,
            "cfg": CFGDIR + '/' + uid + '.d/'}


def statefull(t, num, e):
    uid = t + '-' + str(num)
    return {"tag": t, "type": uid, "id": uid,
            "ip": ip, "port": 6000 + num, "exe": e,
            "vol": DATADIR + '/' + t + '-' + str(num),
            "cfg": CFGDIR + '/' + uid + '.d/'}


def generate_certificate(path):
    # FIXME(jfsmig): Make better than the self-signed certificate here-below
    with open(path + '/cert.csr', 'w') as f:
        subprocess.check_call(('openssl', 'req', '-new'), stdout=f)
    subprocess.check_call(('openssl', 'rsa', '-in', path + '/privkey.pem','-out', path + 'key.pem'))
    subprocess.check_call(('openssl', 'x509', '-in', path + '/cert.csr', '-out', path + '/cert.pem', '-req', '-signkey', 'key.pem', '-days', '1001'))
    with open(path + '/cert.pem', 'a') as f:
        subprocess.check_call(('cat', path + '/key.pem'), stdout=f)


def sequence(start):
    while True:
        yield start
        start+=1


def services():
    port = sequence(0)
    for _i in range(11):
        yield "grpc", statefull("gkindex-store", next(port), "gunkan-index-store-rocksdb")
    for _i in range(11):
        yield "http", statefull("gkblob-store", next(port), "gunkan-blob-store-fs")
    for _i in range(7):
        yield "grpc", stateless("gkindex-gate", next(port), "gunkan-index-gate")
    for _i in range(5):
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
        #generate_certificate(srv['cfg'])
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

# Start the services
children = list()
for kind, srv in services():
    endpoint = srv['ip'] + ':' + str(srv['port'])
    cmd = ('/bin/false',)
    if srv['vol']:  # statefull
        cmd = (srv['exe'], endpoint, srv['vol'])
    else:  # stateless
        cmd = (srv['exe'], endpoint)
    print repr(cmd)
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
